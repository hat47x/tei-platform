export function shouldSubmitXeadPath(
  event: Pick<KeyboardEvent, "key" | "isComposing" | "keyCode">,
): boolean {
  return event.key === "Enter" && !event.isComposing && event.keyCode !== 229;
}

export function handleXeadPathKeydown(
  event: Pick<KeyboardEvent, "key" | "isComposing" | "keyCode" | "preventDefault">,
  observe: () => void,
): boolean {
  if (!shouldSubmitXeadPath(event)) return false;
  event.preventDefault();
  observe();
  return true;
}
