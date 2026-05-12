*** Settings ***
Documentation    Keywords for interacting with the Sparrow Connect-RPC API.
Library          RequestsLibrary
Library          Collections

*** Keywords ***
Create Sparrow Session
    [Documentation]    Create an HTTP session pointing at the Sparrow server.
    [Arguments]    ${url}
    &{headers}=    Create Dictionary    Content-Type=application/json
    Create Session    sparrow    ${url}    headers=${headers}    verify=${False}

Register Event
    [Documentation]    Register an event type.
    [Arguments]    ${event_name}
    ${body}=    Create Dictionary    name=${event_name}    active=${True}
    ${resp}=    POST On Session    sparrow    /webhook.EventService/RegisterEvent
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Push Event
    [Documentation]    Push an event. Returns response with event_id.
    [Arguments]    ${event_name}    ${namespace}    ${payload}    ${idempotency_key}=${NONE}
    ${body}=    Create Dictionary    event=${event_name}    namespace=${namespace}    payload=${payload}
    IF    $idempotency_key is not None
        Set To Dictionary    ${body}    id=${idempotency_key}
    END
    ${resp}=    POST On Session    sparrow    /webhook.EventService/PushEvent
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Register Webhook
    [Documentation]    Register a webhook endpoint with auto-subscriptions. Returns response with webhook_id.
    [Arguments]    ${namespace}    ${url}    @{events}    ${max_retries}=3    ${request_timeout}=10    ${webhook_secret}=e2e-test-secret
    ${http_config}=    Create Dictionary    max_retries=${max_retries}    request_timeout_seconds=${request_timeout}    webhook_secret=${webhook_secret}
    ${body}=    Create Dictionary    namespace=${namespace}    url=${url}    active=${True}    events=${events}    http_config=${http_config}
    ${resp}=    POST On Session    sparrow    /webhook.WebhookService/RegisterWebhook
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Subscribe To Event
    [Documentation]    Create a subscription for a webhook to an event.
    [Arguments]    ${webhook_id}    ${event_name}    ${namespace}    ${template}=${NONE}
    ${body}=    Create Dictionary    webhook_id=${webhook_id}    event_name=${event_name}    namespace=${namespace}
    IF    $template is not None
        Set To Dictionary    ${body}    transform_enabled=${True}    transform_template=${template}
    END
    ${resp}=    POST On Session    sparrow    /webhook.SubscriptionService/CreateSubscription
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

List Deliveries
    [Documentation]    List deliveries with optional filters.
    [Arguments]    ${namespace}=${NONE}    ${webhook_id}=${NONE}    ${event_id}=${NONE}    ${status}=${NONE}
    ${body}=    Create Dictionary
    IF    $namespace is not None
        Set To Dictionary    ${body}    namespace=${namespace}
    END
    IF    $webhook_id is not None
        Set To Dictionary    ${body}    webhook_id=${webhook_id}
    END
    IF    $event_id is not None
        Set To Dictionary    ${body}    event_id=${event_id}
    END
    IF    $status is not None
        Set To Dictionary    ${body}    status=${status}
    END
    ${resp}=    POST On Session    sparrow    /webhook.DeliveryService/ListDeliveries
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Retry Delivery
    [Documentation]    Manually retry a failed delivery.
    [Arguments]    ${delivery_id}
    ${body}=    Create Dictionary    delivery_id=${delivery_id}
    ${resp}=    POST On Session    sparrow    /webhook.DeliveryService/RetryDelivery
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Replay Event
    [Documentation]    Re-push an event (replay to current subscribers).
    [Arguments]    ${event_id}
    ${body}=    Create Dictionary    event_id=${event_id}
    ${resp}=    POST On Session    sparrow    /webhook.EventService/RePushEvent
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Pause Webhook
    [Documentation]    Pause a webhook (stops deliveries).
    [Arguments]    ${webhook_id}    ${namespace}
    ${body}=    Create Dictionary    webhook_id=${webhook_id}    namespace=${namespace}
    ${resp}=    POST On Session    sparrow    /webhook.WebhookService/PauseWebhook
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Resume Webhook
    [Documentation]    Resume a paused webhook.
    [Arguments]    ${webhook_id}    ${namespace}
    ${body}=    Create Dictionary    webhook_id=${webhook_id}    namespace=${namespace}
    ${resp}=    POST On Session    sparrow    /webhook.WebhookService/ResumeWebhook
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Get Delivery Attempts
    [Documentation]    Get all attempts for a delivery.
    [Arguments]    ${delivery_id}
    ${body}=    Create Dictionary    delivery_id=${delivery_id}
    ${resp}=    POST On Session    sparrow    /webhook.DeliveryService/GetDeliveryAttempts
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Get Delivery Status
    [Documentation]    Get full details of a single delivery.
    [Arguments]    ${delivery_id}
    ${body}=    Create Dictionary    delivery_id=${delivery_id}
    ${resp}=    POST On Session    sparrow    /webhook.DeliveryService/GetDeliveryStatus
    ...    json=${body}    expected_status=200
    RETURN    ${resp.json()}

Wait For Delivery Terminal Status
    [Documentation]    Poll a delivery until it reaches a terminal status (SUCCESS, FAILED, EXPIRED).
    [Arguments]    ${delivery_id}    ${timeout}=60s
    ${timeout_sec}=    Evaluate    int("${timeout}".rstrip("s"))
    ${deadline}=    Evaluate    __import__('time').time() + ${timeout_sec}
    WHILE    True
        ${resp}=    Get Delivery Status    ${delivery_id}
        ${status}=    Set Variable    ${resp}[delivery][status]
        IF    "${status}" in ["DELIVERY_SUCCESS", "DELIVERY_FAILED", "DELIVERY_EXPIRED"]
            RETURN    ${resp}[delivery]
        END
        ${now}=    Evaluate    __import__('time').time()
        IF    ${now} >= ${deadline}
            Fail    Delivery ${delivery_id} did not reach terminal status within ${timeout}. Current: ${status}
        END
        Sleep    1s
    END

Wait For All Deliveries Terminal
    [Documentation]    Wait until all deliveries for a namespace reach truly terminal status.
    ...    A delivery is truly terminal when: SUCCESS, EXPIRED, or FAILED with
    ...    attemptCount >= maxAttempts (or non-retryable errorCategory).
    [Arguments]    ${namespace}    ${expected_count}=1    ${timeout}=60s
    ${timeout_sec}=    Evaluate    int("${timeout}".rstrip("s"))
    ${deadline}=    Evaluate    __import__('time').time() + ${timeout_sec}
    @{non_retryable}=    Create List    client_error    dns_error    tls_error    unexpected_status
    WHILE    True
        ${resp}=    List Deliveries    namespace=${namespace}
        ${deliveries}=    Set Variable    ${resp}[deliveries]
        ${all_terminal}=    Set Variable    ${True}
        FOR    ${d}    IN    @{deliveries}
            ${status}=    Set Variable    ${d}[status]
            IF    "${status}" == "DELIVERY_SUCCESS" or "${status}" == "DELIVERY_EXPIRED"
                CONTINUE
            END
            IF    "${status}" == "DELIVERY_FAILED"
                # Check if truly terminal: non-retryable category OR attempts exhausted
                ${cat}=    Set Variable    ${d.get('errorCategory', '')}
                ${attempts}=    Set Variable    ${d.get('attemptCount', 0)}
                ${max}=    Set Variable    ${d.get('maxAttempts', 1)}
                IF    "${cat}" in ${non_retryable} or ${attempts} >= ${max}
                    CONTINUE
                END
            END
            ${all_terminal}=    Set Variable    ${False}
            BREAK
        END
        ${count}=    Get Length    ${deliveries}
        IF    ${all_terminal} and ${count} >= ${expected_count}
            RETURN    ${deliveries}
        END
        ${now}=    Evaluate    __import__('time').time()
        IF    ${now} >= ${deadline}
            Fail    Not all deliveries terminal within ${timeout}. Count: ${count}/${expected_count}
        END
        Sleep    2s
    END

List Events With API Key
    [Documentation]    Call ListEvents with a specific API key header.
    [Arguments]    ${api_key}=${NONE}
    &{headers}=    Create Dictionary    Content-Type=application/json
    IF    $api_key is not None
        Set To Dictionary    ${headers}    X-API-Key=${api_key}
    END
    ${body}=    Create Dictionary
    ${resp}=    POST On Session    sparrow    /webhook.EventService/ListEvents
    ...    json=${body}    headers=${headers}    expected_status=any
    RETURN    ${resp}
