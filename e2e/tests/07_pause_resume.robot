*** Settings ***
Documentation    Scenario 7: Paused Webhook Skips Delivery
...
...    I'm migrating Twilio's webhook endpoint to a new URL. I pause it, push events
...    during the migration window, then resume. Events pushed while paused should
...    not be delivered, but new events after resume should work fine.

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
Paused Webhook Does Not Receive Events
    [Documentation]    Events pushed while webhook is paused are not delivered. Resume restores delivery.
    [Tags]    pause    resume    lifecycle

    ${ns}=    Generate Namespace    pause

    ${twilio_url}=    Start Webhook Target    twilio    ok

    Register Event    sms.received

    ${wh_resp}=    Register Webhook    ${ns}    ${twilio_url}    sms.received
    ${webhook_id}=    Set Variable    ${wh_resp}[webhookId]

    # Baseline: push event, Twilio should receive it
    ${payload1}=    Evaluate    {"from": "+1555000111", "body": "Hello"}
    Push Event    sms.received    ${ns}    ${payload1}
    Wait For Deliveries    twilio    1    timeout=30s

    # Pause the webhook
    Pause Webhook    ${webhook_id}    ${ns}

    # Push event while paused
    ${payload2}=    Evaluate    {"from": "+1555000222", "body": "During maintenance"}
    Push Event    sms.received    ${ns}    ${payload2}

    # Wait and verify nothing new arrived
    Sleep    5s
    ${count}=    Get Delivery Count    twilio
    Should Be Equal As Integers    ${count}    1    Twilio should still have only 1 delivery (paused)

    # Resume the webhook
    Resume Webhook    ${webhook_id}    ${ns}

    # Push event after resume
    ${payload3}=    Evaluate    {"from": "+1555000333", "body": "After maintenance"}
    Push Event    sms.received    ${ns}    ${payload3}
    Wait For Deliveries    twilio    2    timeout=30s

    # Verify Twilio has exactly 2 deliveries (baseline + post-resume, NOT the paused one)
    ${count}=    Get Delivery Count    twilio
    Should Be Equal As Integers    ${count}    2
