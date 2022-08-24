import { useState } from 'react'

export default function SignupRedirect() {
  const [error, setError] = useState('')

  async function signupRedirect() {
    try {
      // set the org auth cookies
      let res = await fetch('/api/signup/session')

      if (!res.ok) {
        throw await res.json()
      }

      // redirect to destinations
      window.location = `${window.location.protocol}//${window.location.hostname}/destinations`
    } catch (e) {
      setError(e.message)
    }
  }

  signupRedirect()

  return (
    <div className='flex h-full w-full items-center justify-center'>
      {error === '' ? (
        <img
          alt='loading'
          className='h-20 w-20 animate-spin-fast'
          src='/spinner.svg'
        />
      ) : (
        <p className='text-s mx-auto w-full text-center text-pink-400'>
          error: {error}
        </p>
      )}
    </div>
  )
}
