import { useState, useRef } from 'react'
import ReCAPTCHA from 'react-google-recaptcha'

export default function () {
  const [email, setEmail] = useState('')
  const [error, setError] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const recaptchaRef = useRef()

  async function onReCAPTCHAChange (code) {
    if (!code) {
      return
    }

    try {
      await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({ email, code })
      })
      setSubmitted(true)
    } catch (e) {
      setError(true)
    }

    if (window.analytics) {
      window.analytics.identify({
        email
      })
    }
  }

  async function onSubmit (e) {
    e.preventDefault()
    recaptchaRef.current.execute()
    return false
  }

  if (submitted) {
    return (
      <p className='flex items-center text-lg ml-4 text-white'>
        You're on the list. We'll be in touch!
      </p>
    )
  }

  return (
    <div className='flex flex-col flex-1 relative'>
      <ReCAPTCHA
        ref={recaptchaRef}
        size='invisible'
        sitekey='6Lcld3EcAAAAAONnvAUZR6igONL-TZm9XextIS9U'
        onChange={onReCAPTCHAChange}
      />
      <form className='flex flex-1 relative rounded-full border-2 border-zinc-800' onSubmit={onSubmit}>
        <input
          type='email'
          required
          className='flex-1 bg-transparent w-full py-2 text-lg pl-6 pr-36 rounded-full'
          placeholder='email'
          onChange={e => {
            setEmail(e.target.value)
            setError(false)
          }}
        />
        <input type='submit' className='cursor-pointer absolute top-0 bottom-0 right-0 text-gray-300 hover:text-white px-6 rounded-full bg-zinc-900 flex-none' value='Get Updates' />
      </form>
      {error && <div className='text-sm text-red-400 mt-2 ml-6 -bottom-8 absolute'>Could not register for updates</div>}
    </div>
  )
}
