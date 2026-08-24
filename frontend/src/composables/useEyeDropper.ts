// Wraps the native EyeDropper API (Chrome/Edge 95+), letting the user pick a
// color from anywhere on screen — including the live preview — instead of
// only from the OS color picker. Not supported in Firefox/Safari as of 2026;
// callers should hide the eyedropper button when isEyeDropperSupported is false.
export const isEyeDropperSupported = typeof window !== 'undefined' && 'EyeDropper' in window

export async function pickColorFromScreen(): Promise<string | null> {
  if (!isEyeDropperSupported) return null
  try {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const eyeDropper = new (window as any).EyeDropper()
    const result = await eyeDropper.open()
    return result.sRGBHex as string
  } catch {
    // User pressed Escape / cancelled — not an error.
    return null
  }
}
