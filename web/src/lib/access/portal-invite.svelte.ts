// Portal invite redemption state, shown by the portal page while a
// /portal#invite=... link is being exchanged for a consumer token.
let status = $state<"idle" | "redeeming" | "failed">("idle");
let message = $state("");

export const portalInvite = {
  get status() {
    return status;
  },
  get message() {
    return message;
  },
  start() {
    status = "redeeming";
  },
  fail(text: string) {
    status = "failed";
    message = text;
  },
};
