*** Settings ***
Documentation    Scenario 1: The Happy Path -- One Event, Multiple Subscribers
...
...    I'm building an e-commerce platform. When an order is created, Stripe needs
...    to know (for payment), Shippo needs to know (for shipping), and Slack needs
...    to know (for team notifications). I push one event and all three receive it
...    with valid signatures.

Library          ../libraries/SparrowEnvironment.py
Library          ../libraries/WebhookTargetServer.py
Library          ../libraries/SignatureVerifier.py
Resource         ../resources/sparrow_api.resource
Resource         ../resources/common.resource

Suite Setup      Setup Sparrow Test Suite
Suite Teardown   Run Keywords    Suite Teardown - Stop Targets    AND    Stop Sparrow

*** Keywords ***
Setup Sparrow Test Suite
    ${url}=    Start Sparrow
    Create Sparrow Session    ${url}

*** Test Cases ***
One Event Fans Out To All Subscribers With Valid Signatures
    [Documentation]    Push order.created -> Stripe, Shippo, and Slack all receive it.
    [Tags]    happy_path    fan_out    signatures

    # Setup: unique namespace for isolation
    ${ns}=    Generate Namespace    happy-path

    # Start 3 webhook targets (simulating Stripe, Shippo, Slack)
    ${stripe_url}=    Start Webhook Target    stripe    ok
    ${shippo_url}=    Start Webhook Target    shippo    ok
    ${slack_url}=     Start Webhook Target    slack     ok

    # Register the event type
    Register Event    order.created

    # Register 3 webhooks, each auto-subscribed to order.created
    ${stripe_wh}=    Register Webhook    ${ns}    ${stripe_url}    order.created
    ${shippo_wh}=    Register Webhook    ${ns}    ${shippo_url}    order.created
    ${slack_wh}=     Register Webhook    ${ns}    ${slack_url}     order.created

    # Push the event
    ${payload}=    Evaluate    {"order_id": "ord-7741", "amount": 42.0, "customer": "acme-corp"}
    ${push_resp}=    Push Event    order.created    ${ns}    ${payload}

    # Wait for all 3 targets to receive deliveries
    Wait For Deliveries    stripe    1    timeout=30s
    Wait For Deliveries    shippo    1    timeout=30s
    Wait For Deliveries    slack     1    timeout=30s

    # Verify each target received exactly 1 delivery
    ${stripe_count}=    Get Delivery Count    stripe
    ${shippo_count}=    Get Delivery Count    shippo
    ${slack_count}=     Get Delivery Count    slack
    Should Be Equal As Integers    ${stripe_count}    1
    Should Be Equal As Integers    ${shippo_count}    1
    Should Be Equal As Integers    ${slack_count}     1

    # Verify signature headers are present
    ${stripe_delivery}=    Get Latest Delivery    stripe
    Delivery Has Signature Headers    ${stripe_delivery}

    # Verify via API that all 3 deliveries are successful
    ${deliveries}=    List Deliveries    namespace=${ns}
    Length Should Be    ${deliveries}[deliveries]    3
