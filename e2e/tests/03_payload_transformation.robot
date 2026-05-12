*** Settings ***
Documentation    Scenario 3: Payload Transformation via Template
...
...    I want to forward alerts to Slack, but Slack expects a specific JSON format
...    with a 'text' field. I set up a subscription with a Go template that transforms
...    my alert payload into Slack's format.

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
Template Transforms Payload Before Delivery
    [Documentation]    Slack receives a transformed payload, not the default envelope.
    [Tags]    template    transformation

    ${ns}=    Generate Namespace    transform

    # Start Slack target
    ${slack_url}=    Start Webhook Target    slack    ok

    # Register event
    Register Event    alert.fired

    # Register webhook (no auto-subscriptions -- we'll create one manually with a template)
    ${wh_resp}=    Register Webhook    ${ns}    ${slack_url}
    ${webhook_id}=    Set Variable    ${wh_resp}[webhookId]

    # Subscribe with a Go template that outputs Slack's format
    ${template}=    Set Variable    {"text": "Alert: {{.payload.title}} - severity {{.payload.severity}}"}
    Subscribe To Event    ${webhook_id}    alert.fired    ${ns}    template=${template}

    # Push the event
    ${payload}=    Evaluate    {"title": "CPU High on prod-api-3", "severity": "critical", "host": "prod-api-3"}
    Push Event    alert.fired    ${ns}    ${payload}

    # Wait for delivery
    Wait For Deliveries    slack    1    timeout=30s

    # Verify Slack received the transformed payload (NOT the envelope)
    ${delivery}=    Get Latest Delivery    slack
    ${body}=    Set Variable    ${delivery}[body]
    Should Be Equal    ${body}[text]    Alert: CPU High on prod-api-3 - severity critical

    # Verify it's NOT the envelope format (no "version" or "event_name" keys)
    Dictionary Should Not Contain Key    ${body}    version
    Dictionary Should Not Contain Key    ${body}    event_name
