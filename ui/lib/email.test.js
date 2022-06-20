import { validateEmail } from './email'

test('validates email', () => {
  expect(validateEmail('example@example.com')).toBe(true)
})

test('rejects non-email', () => {
  expect(validateEmail('example')).toBe(false)
})
