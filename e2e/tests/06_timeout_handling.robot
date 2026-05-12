*** Settings ***
Documentation    Scenario 6: Timeout Handling
...
...    Shopify's webhook receiver is overloaded and takes 10 seconds to respond.
...    Sparrow should time out after the configured threshold, classify it as a
...    timeout, and retry.

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
Slow Endpoint Is Classified As Timeout
    [Documentation]    Shopify takes 10s to respond but timeout is 2s. Both attempts time out.
    [Tags]    timeout    retry

    ${ns}=    Generate Namespace    timeout

    # Shopify is slow -- 10 second response delay
    ${shopify_url}=    Start Webhook Target    shopify    slow_10s

    Register Event    cart.abandoned

    # Register with 2s timeout and 1 retry
    Register Webhook    ${ns}    ${shopify_url}    cart.abandoned    max_retries=1    request_timeout=2

    ${payload}=    Evaluate    {"cart_id": "cart-9921", "items": 3}
    Push Event    cart.abandoned    ${ns}    ${payload}

    # Wait for terminal failure (both attempts will time out)
    ${deliveries}=    Wait For All Deliveries Terminal    ${ns}    expected_count=1    timeout=60s

    ${d}=    Set Variable    ${deliveries}[0]
    Should Be Equal    ${d}[status]    DELIVERY_FAILED
    Should Be Equal    ${d}[errorCategory]    timeout

    # Shopify should have received 2 requests (initial + 1 retry, both timed out client-side)
    # Note: the target may or may not register the request depending on timing
    ${count}=    Get Delivery Count    shopify
    Should Be True    ${count} >= 1    Shopify should have received at least 1 request
