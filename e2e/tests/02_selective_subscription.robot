*** Settings ***
Documentation    Scenario 2: Selective Subscription -- Only Matching Subscribers Get Hit
...
...    I'm running a marketplace. Stripe subscribes to order.created (for charging),
...    FedEx subscribes to order.shipped (for logistics), and Datadog subscribes to
...    both (for observability). When an order is created, only Stripe and Datadog
...    should get the webhook -- FedEx has nothing to do with it.

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
Only Matching Subscribers Receive Deliveries
    [Documentation]    Push order.created -> Stripe and Datadog get it, FedEx does not.
    [Tags]    subscription    filtering

    ${ns}=    Generate Namespace    selective

    # Start 3 targets
    ${stripe_url}=     Start Webhook Target    stripe     ok
    ${fedex_url}=      Start Webhook Target    fedex      ok
    ${datadog_url}=    Start Webhook Target    datadog    ok

    # Register both event types
    Register Event    order.created
    Register Event    order.shipped

    # Stripe subscribes to order.created only
    Register Webhook    ${ns}    ${stripe_url}     order.created

    # FedEx subscribes to order.shipped only
    Register Webhook    ${ns}    ${fedex_url}      order.shipped

    # Datadog subscribes to both
    Register Webhook    ${ns}    ${datadog_url}    order.created    order.shipped

    # Push order.created
    ${payload}=    Evaluate    {"order_id": "ord-501", "total": 129.99}
    Push Event    order.created    ${ns}    ${payload}

    # Stripe and Datadog should each receive 1 delivery
    Wait For Deliveries    stripe     1    timeout=30s
    Wait For Deliveries    datadog    1    timeout=30s

    # FedEx should receive nothing
    Assert No Deliveries    fedex    wait=5s

    # Verify counts
    ${stripe_count}=     Get Delivery Count    stripe
    ${fedex_count}=      Get Delivery Count    fedex
    ${datadog_count}=    Get Delivery Count    datadog
    Should Be Equal As Integers    ${stripe_count}     1
    Should Be Equal As Integers    ${fedex_count}      0
    Should Be Equal As Integers    ${datadog_count}    1

    # API should show exactly 2 deliveries
    ${deliveries}=    List Deliveries    namespace=${ns}
    Length Should Be    ${deliveries}[deliveries]    2
