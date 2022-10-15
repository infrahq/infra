import { useState, useRef } from 'react'
import ReCAPTCHA from 'react-google-recaptcha'

export function useSignup({ email }) {
  const [error, setError] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [success, setSuccess] = useState(false)
  const ref = useRef()

  async function onReCAPTCHAChange(code) {
    if (!code) {
      return
    }

    try {
      const res = await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({
          email,
          code,
          aid: window?.analytics.user().anonymousId(),
        }),
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
      })

      if (!res.ok) {
        throw await res.json()
      }
    } catch (e) {
      setError(true)
      return
    }

    setSubmitted(false)
    setSuccess(true)
  }

  return {
    submitted,
    success,
    error,
    submit: function () {
      setSubmitted(true)
      setError(false)
      if (ref.current) {
        ref.current.execute()
      }
      return false
    },
    renderRecaptcha: function () {
      return (
        <ReCAPTCHA
          ref={ref}
          size='invisible'
          sitekey='6Lcld3EcAAAAAONnvAUZR6igONL-TZm9XextIS9U'
          onChange={onReCAPTCHAChange}
        />
      )
    },
  }
}
