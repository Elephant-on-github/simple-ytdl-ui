import { describe, expect, it } from 'vitest'
import { describeOAuthError } from './oauth'

describe('describeOAuthError', () => {
  it('turns a known code into a readable message', () => {
    expect(describeOAuthError('invalid_state')).toContain('expired')
    expect(describeOAuthError('exchange_failed')).toContain('access token')
  })

  it('falls back to a generic message for an unknown code', () => {
    expect(describeOAuthError('something_new')).toContain('try again')
  })

  it('says nothing when there is no code', () => {
    expect(describeOAuthError('')).toBe('')
  })
})
