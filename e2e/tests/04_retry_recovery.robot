*** Settings ***
Documentation    Scenario 4: Target Failures and Retry Recovery
...
...    Google's Cloud Functions endpoint is having a bad deployment -- it returns 500
...    for a few minutes, then recovers. Sparrow should keep retrying and eventually
...    deliver successfully once Google's endpoint is back.

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
Retries Succeed After Target Recovers
    [Documentation]    Google's endpoint fails twice then recovers. Sparrow retries and delivers.
    [Tags]    retry    recovery

    ${ns}=    Generate Namespace    retry

    # Google fails for first 1 request, then 200 on 2nd attempt
    # River's backoff is ~16s for 2nd attempt, so keep timeouts generous
    ${google_url}=    Start Webhook Target    google    fail_then_succeed_1

    Register Event    user.signup

    Register Webhook    ${ns}    ${google_url}    user.signup    max_retries=3

    # Push event
    ${payload}=    Evaluate    {"user_id": "usr-8821", "email": "jane@example.com"}
    Push Event    user.signup    ${ns}    ${payload}

    # Wait for the successful delivery (after retry -- River backoff ~16s for attempt 2)
    ${deliveries}=    Wait For All Deliveries Terminal    ${ns}    expected_count=1    timeout=90s

    # Verify final status is success
    ${d}=    Set Variable    ${deliveries}[0]
    Should Be Equal    ${d}[status]    DELIVERY_SUCCESS

    # Verify attempt count = 2 (1 failure + 1 success)
    Should Be Equal As Integers    ${d}[attemptCount]    2

    # Google's endpoint should have received 2 requests total
    ${count}=    Get Delivery Count    google
    Should Be Equal As Integers    ${count}    2

    # Verify attempts via API
    ${attempts_resp}=    Get Delivery Attempts    ${d}[deliveryId]
    ${attempts}=    Set Variable    ${attempts_resp}[attempts]
    Length Should Be    ${attempts}    2
