*** Settings ***
Documentation    Scenario 10: Invalid Template Graceful Degradation
...
...    An engineer wrote a bad Go template for the PagerDuty subscription -- it
...    references a field that doesn't exist. When an event fires, PagerDuty should
...    still get the delivery (with the raw envelope payload as fallback) rather than
...    silently dropping it.

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
Broken Template Falls Back To Envelope Payload
    [Documentation]    PagerDuty receives the envelope (fallback) when template execution fails.
    [Tags]    template    fallback    graceful_degradation

    ${ns}=    Generate Namespace    fallback

    ${pagerduty_url}=    Start Webhook Target    pagerduty    ok

    Register Event    deploy.failed

    # Register webhook without auto-subscription
    ${wh_resp}=    Register Webhook    ${ns}    ${pagerduty_url}
    ${webhook_id}=    Set Variable    ${wh_resp}[webhookId]

    # Subscribe with a BROKEN template that will ERROR during execution
    # (calling "index" on a nil value panics in Go templates)
    ${bad_template}=    Set Variable    {{index .payload.nonexistent "key"}}
    Subscribe To Event    ${webhook_id}    deploy.failed    ${ns}    template=${bad_template}

    # Push event
    ${payload}=    Evaluate    {"service": "api-gateway", "version": "v2.3.1", "error": "health check timeout"}
    Push Event    deploy.failed    ${ns}    ${payload}

    # PagerDuty should still receive a delivery (not dropped!)
    Wait For Deliveries    pagerduty    1    timeout=30s

    # Verify it received the envelope format (fallback)
    ${delivery}=    Get Latest Delivery    pagerduty
    ${body}=    Set Variable    ${delivery}[body]
    Dictionary Should Contain Key    ${body}    version
    Dictionary Should Contain Key    ${body}    event_name
    Dictionary Should Contain Key    ${body}    payload
    Should Be Equal    ${body}[event_name]    deploy.failed

    # Verify delivery succeeded via API
    ${deliveries}=    Wait For All Deliveries Terminal    ${ns}    expected_count=1    timeout=30s
    ${d}=    Set Variable    ${deliveries}[0]
    Should Be Equal    ${d}[status]    DELIVERY_SUCCESS
