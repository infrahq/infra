import { useState, useRef } from 'react'
import ReCAPTCHA from 'react-google-recaptcha'
import analytics from '../lib/analytics'

export default function SignupForm() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const recaptchaRef = useRef()

  async function onReCAPTCHAChange(code) {
    if (!code) {
      return
    }

    const user = await analytics.user()

    try {
      const res = await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({
          email,
          code,
          aid: user.anonymousId(),
        }),
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
      })

      if (!res.ok) {
        throw res
      }

      setSubmitted(true)
    } catch (e) {
      setError(true)
      console.error(e)
    }
  }

  async function onSubmit(e) {
    e.preventDefault()
    recaptchaRef.current.execute()
    return false
  }

  if (submitted) {
    return (
      <p className='ml-4 flex items-center'>
        {`You're on the list. We'll be in touch!`}
      </p>
    )
  }

  return (
    <div className='relative flex flex-1 flex-col'>
      <ReCAPTCHA
        ref={recaptchaRef}
        size='invisible'
        sitekey='6Lcld3EcAAAAAONnvAUZR6igONL-TZm9XextIS9U'
        onChange={onReCAPTCHAChange}
        onError={e => console.log(e)}
      />
      <form className='relative flex flex-1 space-x-2' onSubmit={onSubmit}>
        <input
          id='email-address'
          name='email'
          type='email'
          autoComplete='email'
          required
          className='w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm leading-4 placeholder-gray-500 focus:border-blue-500 focus:ring-blue-500'
          placeholder='your email'
          onChange={e => setEmail(e.target.value)}
        />
        <input
          type='submit'
          className='inline-flex items-center rounded-lg bg-black px-3 py-2 text-sm font-semibold tracking-tight text-white hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'
          value='Sign Up'
        />
      </form>
      {error && (
        <div className='absolute -bottom-8 mt-5 ml-6 text-xs text-red-400'>
          Could not register for updates
        </div>
      )}
    </div>
  )
}
