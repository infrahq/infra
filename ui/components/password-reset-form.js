import { useRouter } from 'next/router'
import { useState } from 'react'
import { useSWRConfig } from 'swr'

export default function PasswordResetForm() {
  const { mutate } = useSWRConfig()
  const router = useRouter()
  const { token } = router.query

  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function onSubmit(e) {
    e.preventDefault()

    setSubmitting(true)

    try {
      const res = await fetch('/api/password-reset', {
        method: 'post',
        body: JSON.stringify({
          token,
          password,
        }),
      })

      await jsonBody(res)

      await mutate('/api/users/self')
      router.replace('/')
    } catch (e) {
      setSubmitting(false)
      setError(e.message)
    }

    return false
  }

  return (
    <form
      onSubmit={onSubmit}
      className='relative flex w-full max-w-sm flex-col'
    >
      <div className='my-2 w-full'>
        <label htmlFor='name' className='text-2xs font-medium text-gray-700'>
          Password
        </label>
        <input
          required
          autoFocus
          name='password'
          type='password'
          onChange={e => {
            setPassword(e.target.value)
            setError('')
          }}
          className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
            error ? 'border-red-500' : 'border-gray-300'
          }`}
        />
      </div>
      <button
        disabled={submitting}
        className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 sm:text-sm'
      >
        Set Password
      </button>
      {error && (
        <p className='absolute top-full mx-auto w-full text-center text-2xs text-red-500'>
          {error}
        </p>
      )}
    </form>
  )
}
