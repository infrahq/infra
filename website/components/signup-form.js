import { useState, useRef } from 'react'
import ReCAPTCHA from 'react-google-recaptcha'

export default function SignupForm() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const recaptchaRef = useRef()

  async function onReCAPTCHAChange(code) {
    if (!code) {
      return
    }

    try {
      const res = await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({ email, code }),
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
    }

    if (window.analytics) {
      window.analytics.identify({
        email,
      })
    }
  }

  async function onSubmit(e) {
    e.preventDefault()
    recaptchaRef.current.execute()
    return false
  }

  if (submitted) {
    return (
      <p className='ml-4 flex items-center text-lg text-white'>
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
      />
      <form
        className='relative flex flex-1 rounded-full border-2 border-zinc-800'
        onSubmit={onSubmit}
      >
        <input
          type='email'
          required
          className='w-full flex-1 rounded-full bg-transparent py-2 pl-6 pr-36 text-lg'
          placeholder='email'
          onChange={e => {
            setEmail(e.target.value)
            setError(false)
          }}
        />
        <input
          type='submit'
          className='absolute top-0 bottom-0 right-0 flex-none cursor-pointer rounded-full bg-zinc-900 px-6 text-gray-300 hover:text-white'
          value='Get Updates'
        />
      </form>
      {error && (
        <div className='absolute -bottom-8 mt-2 ml-6 text-sm text-red-400'>
          Could not register for updates
        </div>
      )}
    </div>
  )
}
