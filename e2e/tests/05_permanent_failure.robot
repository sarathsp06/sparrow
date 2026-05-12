*** Settings ***
Documentation    Scenario 5: Permanent Failure -- Target Always Broken
...
...    Facebook's webhook endpoint has been decommissioned and returns 404. Sparrow
...    should NOT waste retries on a client error. Separately, Microsoft's endpoint
...    returns 500 forever, exhausting all retries.

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
Client Error Is Not Retried
    [Documentation]    Facebook returns 404 -- no retries, immediate failure.
    [Tags]    failure    client_error

    ${ns}=    Generate Namespace    perm-fail-a

    ${facebook_url}=    Start Webhook Target    facebook    status_404

    Register Event    payment.refunded

    Register Webhook    ${ns}    ${facebook_url}    payment.refunded    max_retries=3

    ${payload}=    Evaluate    {"payment_id": "pay-112", "amount": 25.00}
    Push Event    payment.refunded    ${ns}    ${payload}

    # Wait for terminal status
    ${deliveries}=    Wait For All Deliveries Terminal    ${ns}    expected_count=1    timeout=30s

    ${d}=    Set Variable    ${deliveries}[0]
    Should Be Equal    ${d}[status]    DELIVERY_FAILED
    Should Be Equal    ${d}[errorCategory]    client_error
    Should Be Equal As Integers    ${d}[attemptCount]    1

    # Facebook got exactly 1 request (no retries)
    ${count}=    Get Delivery Count    facebook
    Should Be Equal As Integers    ${count}    1

Server Error Exhausts All Retries
    [Documentation]    Microsoft returns 500 forever -- retries exhaust then fails.
    [Tags]    failure    server_error

    ${ns}=    Generate Namespace    perm-fail-b

    ${microsoft_url}=    Start Webhook Target    microsoft    status_500

    Register Event    payment.refunded.v2

    Register Webhook    ${ns}    ${microsoft_url}    payment.refunded.v2    max_retries=2

    ${payload}=    Evaluate    {"payment_id": "pay-113", "amount": 50.00}
    Push Event    payment.refunded.v2    ${ns}    ${payload}

    # Wait for terminal status (initial + 2 retries = 3 attempts)
    ${deliveries}=    Wait For All Deliveries Terminal    ${ns}    expected_count=1    timeout=60s

    ${d}=    Set Variable    ${deliveries}[0]
    Should Be Equal    ${d}[status]    DELIVERY_FAILED
    Should Be Equal    ${d}[errorCategory]    server_error
    Should Be Equal As Integers    ${d}[attemptCount]    3

    # Microsoft got 3 requests (initial + 2 retries)
    ${count}=    Get Delivery Count    microsoft
    Should Be Equal As Integers    ${count}    3
