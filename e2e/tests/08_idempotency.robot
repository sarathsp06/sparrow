*** Settings ***
Documentation    Scenario 8: Idempotency -- Same Event Pushed Twice
...
...    My payment service has at-least-once semantics and might call PushEvent twice
...    for the same charge. I use an idempotency key so Sparrow deduplicates and only
...    delivers once -- Stripe shouldn't charge the customer twice.

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
Duplicate Push With Idempotency Key Delivers Once
    [Documentation]    Pushing the same event twice with the same idempotency key results in 1 delivery.
    [Tags]    idempotency    dedup

    ${ns}=    Generate Namespace    idemp

    ${stripe_url}=    Start Webhook Target    stripe    ok

    Register Event    charge.completed

    Register Webhook    ${ns}    ${stripe_url}    charge.completed

    # First push with idempotency key
    ${payload}=    Evaluate    {"amount": 99.99, "currency": "USD"}
    ${resp1}=    Push Event    charge.completed    ${ns}    ${payload}    idempotency_key=charge-xyz-001
    Should Not Be True    ${resp1.get('duplicate', False)}

    # Second push with SAME idempotency key
    ${resp2}=    Push Event    charge.completed    ${ns}    ${payload}    idempotency_key=charge-xyz-001
    Should Be True    ${resp2}[duplicate]

    # Same event_id returned
    Should Be Equal    ${resp1}[eventId]    ${resp2}[eventId]

    # Wait for delivery and verify only 1
    Wait For Deliveries    stripe    1    timeout=30s
    Sleep    3s    # extra wait to ensure no second delivery arrives
    ${count}=    Get Delivery Count    stripe
    Should Be Equal As Integers    ${count}    1

    # API confirms 1 delivery
    ${deliveries}=    List Deliveries    namespace=${ns}
    Length Should Be    ${deliveries}[deliveries]    1

Push Without Idempotency Key Creates Separate Events
    [Documentation]    Two pushes without idempotency key create 2 separate events and 2 deliveries.
    [Tags]    idempotency    no_dedup

    ${ns}=    Generate Namespace    no-idemp

    ${stripe2_url}=    Start Webhook Target    stripe2    ok

    Register Event    charge.completed.v2

    Register Webhook    ${ns}    ${stripe2_url}    charge.completed.v2

    ${payload}=    Evaluate    {"amount": 50.00}
    ${resp1}=    Push Event    charge.completed.v2    ${ns}    ${payload}
    ${resp2}=    Push Event    charge.completed.v2    ${ns}    ${payload}

    # Different event IDs
    Should Not Be Equal    ${resp1}[eventId]    ${resp2}[eventId]

    # 2 deliveries
    Wait For Deliveries    stripe2    2    timeout=30s
    ${count}=    Get Delivery Count    stripe2
    Should Be Equal As Integers    ${count}    2
