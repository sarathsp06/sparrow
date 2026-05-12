*** Settings ***
Documentation    Common setup and teardown for all Sparrow e2e tests.
Library          ../libraries/WebhookTargetServer.py

*** Keywords ***
Suite Teardown - Stop Targets
    [Documentation]    Stop all mock webhook targets.
    Stop All Targets

Generate Namespace
    [Documentation]    Generate a unique namespace for test isolation.
    [Arguments]    ${prefix}=e2e
    ${ts}=    Evaluate    str(int(__import__('time').time() * 1000) % 100000)
    ${ns}=    Set Variable    ${prefix}-${ts}
    RETURN    ${ns}
