*** Settings ***
Documentation    Scenario 11: Manual Retry and Event Replay
...
...    We pushed an invoice event but Facebook's endpoint was down (404). The delivery
...    failed permanently. After fixing the URL, we manually retry that specific delivery.
...    We also replay the event to pick up Zendesk, a new subscriber added after the push.

Library          ../libraries/SparrowEnvironment.py
Library          ../libraries/WebhookTargetServer.py
Resource         ../resources/sparrow_api.resource
Resource         ../resources/common.resource

Suite Setup      Setup Sparrow Test Suite
Suite Teardown   Run Keywords    Suite Teardown - Stop Targets    AND    Stop Sparrow

*** Keywords ***
Setup Sparrow Test Suite
    ${url}=    Start Sparrow
    Create Sparrow Session    ${url}

*** Test Cases ***
Retry Failed Delivery After Fixing Endpoint
    [Documentation]    Facebook fails with 404, gets fixed, manual retry succeeds.
    [Tags]    retry    manual    replay

    ${ns}=    Generate Namespace    retry-replay

    # Facebook starts broken (404), GitHub works
    ${facebook_url}=    Start Webhook Target    facebook    status_404
    ${github_url}=      Start Webhook Target    github     ok

    Register Event    invoice.sent

    Register Webhook    ${ns}    ${facebook_url}    invoice.sent    max_retries=0
    Register Webhook    ${ns}    ${github_url}      invoice.sent

    # Push event
    ${payload}=    Evaluate    {"invoice_id": "inv-2024-099", "customer": "Acme Corp", "total": 15000.00}
    ${push_resp}=    Push Event    invoice.sent    ${ns}    ${payload}
    ${event_id}=    Set Variable    ${push_resp}[eventId]

    # Wait for both deliveries to reach terminal status
    ${deliveries}=    Wait For All Deliveries Terminal    ${ns}    expected_count=2    timeout=30s

    # Find Facebook's failed delivery and GitHub's success
    ${facebook_delivery}=    Set Variable    ${NONE}
    ${github_delivery}=    Set Variable    ${NONE}
    FOR    ${d}    IN    @{deliveries}
        IF    "${d}[status]" == "DELIVERY_FAILED"
            ${facebook_delivery}=    Set Variable    ${d}
        ELSE IF    "${d}[status]" == "DELIVERY_SUCCESS"
            ${github_delivery}=    Set Variable    ${d}
        END
    END

    Should Not Be Equal    ${facebook_delivery}    ${NONE}    Facebook delivery should have failed
    Should Not Be Equal    ${github_delivery}    ${NONE}    GitHub delivery should have succeeded

    Should Be Equal    ${facebook_delivery}[errorCategory]    client_error
    Should Be Equal As Integers    ${facebook_delivery}[attemptCount]    1

    # Fix Facebook's endpoint
    Switch Target Behavior    facebook    ok

    # Manually retry Facebook's failed delivery
    Retry Delivery    ${facebook_delivery}[deliveryId]

    # Wait for the retry to complete
    ${retried}=    Wait For Delivery Terminal Status    ${facebook_delivery}[deliveryId]    timeout=30s
    Should Be Equal    ${retried}[status]    DELIVERY_SUCCESS

    # Facebook should now have received 2 requests (original 404 + successful retry)
    ${fb_count}=    Get Delivery Count    facebook
    Should Be Equal As Integers    ${fb_count}    2

Replay Event Reaches New Subscribers
    [Documentation]    RePushEvent replays to current subscriptions, including newly added Zendesk.
    [Tags]    replay    repush

    ${ns}=    Generate Namespace    replay

    ${github_url}=    Start Webhook Target    github2    ok

    Register Event    invoice.sent.v2

    Register Webhook    ${ns}    ${github_url}    invoice.sent.v2

    # Push original event
    ${payload}=    Evaluate    {"invoice_id": "inv-2024-100", "customer": "Beta Corp"}
    ${push_resp}=    Push Event    invoice.sent.v2    ${ns}    ${payload}
    ${event_id}=    Set Variable    ${push_resp}[eventId]

    Wait For Deliveries    github2    1    timeout=30s

    # Add a NEW subscriber (Zendesk) AFTER the original push
    ${zendesk_url}=    Start Webhook Target    zendesk    ok
    Register Webhook    ${ns}    ${zendesk_url}    invoice.sent.v2

    # Replay the event
    ${replay_resp}=    Replay Event    ${event_id}

    # Both GitHub and Zendesk should receive the replayed event
    Wait For Deliveries    github2    2    timeout=30s
    Wait For Deliveries    zendesk    1    timeout=30s

    ${zendesk_count}=    Get Delivery Count    zendesk
    Should Be Equal As Integers    ${zendesk_count}    1
