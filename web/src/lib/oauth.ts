/**
 * The Google sign-in hands problems back to the app as a code in the query
 * string. This turns that code into something a person can read.
 */
export const OAUTH_ERRORS: Record<string, string> = {
  invalid_state: 'The sign-in request expired before it was finished. Please try again.',
  missing_code: 'Google did not send an authorization code back, so nothing was signed in.',
  exchange_failed: 'Google did not hand out an access token. Please try again.',
  missing_refresh_token:
    'Google did not hand out a refresh token, so the sign-in cannot be kept. Please try again and allow access to your playlists.',
  storage_failed: 'The sign-in worked, but the server could not save the token.',
}

/** The message for an error code, or a generic one for an unknown code. */
export function describeOAuthError(code: string): string {
  if (!code) return ''
  return (
    OAUTH_ERRORS[code] ??
    'Google sign-in did not work. Please try again — the details are in the server log.'
  )
}
