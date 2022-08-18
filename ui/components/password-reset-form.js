import { useRouter } from 'next/router'
import { useState } from 'react'
import { useSWRConfig } from 'swr'

export default function PasswordResetForm() {
  const { mutate } = useSWRConfig()
  const router = useRouter()
  const { token } = router.query

  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmitSetPassword(e) {
    e.preventDefault()

    try {
      const res = await fetch('/api/password-reset', {
        method: 'post',
        body: JSON.stringify({
          token,
          password,
        }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      await res.json()

      await mutate('/api/users/self')
      router.replace('/')
    } catch (e) {
      setError(e.message)
    }

    return false
  }

  return (
    <form
      onSubmit={onSubmitSetPassword}
      className='relative flex w-full max-w-sm flex-col'
    >
      <div className='my-2 w-full'>
        <label htmlFor='password' className='text-3xs uppercase text-gray-500'>
          Password
        </label>
        <input
          required
          id='password'
          type='password'
          data-testid='form-field-password'
          placeholder='enter your password'
          onChange={e => {
            setPassword(e.target.value)
            setError('')
          }}
          className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
            error ? 'border-pink-500/60' : ''
          }`}
        />
      </div>
      <button
        disabled={!password}
        className='mt-6 mb-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
      >
        Set Password
      </button>
      {error && (
        <p className='absolute -bottom-3.5 mx-auto w-full text-center text-2xs text-pink-400'>
          {error}
        </p>
      )}
    </form>
  )
}
