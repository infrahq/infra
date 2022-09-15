import Head from 'next/head'
import { useState } from 'react'
import useSWR from 'swr'

import ErrorMessage from '../../components/error-message'
import Dashboard from '../../components/layouts/dashboard'
import Notification from '../../components/notification'

function PasswordReset({ onReset = () => {} }) {
  const { data: auth } = useSWR('/api/users/self')

  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const [submitting, setSubmitting] = useState(false)

  async function onSubmit(e) {
    e.preventDefault()

    if (password !== confirmPassword) {
      setErrors({
        confirmPassword: 'passwords do not match',
      })
      return false
    }

    setSubmitting(true)
    setError('')
    setErrors({})

    try {
      const rest = await fetch(`/api/users/${auth?.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          ...auth,
          password: confirmPassword,
        }),
      })

      setSubmitting(false)

      const data = await rest.json()

      if (!rest.ok) {
        throw data
      }

      setPassword('')
      setConfirmPassword('')
      onReset()
    } catch (e) {
      setSubmitting(false)
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] =
            error.errors[0] || 'invalid value'
        }
        setErrors(errors)
      } else {
        setError(e.message)
      }
    }
  }

  return (
    <form onSubmit={onSubmit} className='flex flex-col'>
      <div className='relative my-2 w-full'>
        <label
          htmlFor='password'
          className='text-2xs font-medium text-gray-700'
        >
          New Password
        </label>
        <input
          required
          name='password'
          type='password'
          value={password}
          onChange={e => {
            setPassword(e.target.value)
            setErrors({})
            setError('')
          }}
          className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
            errors.password ? 'border-red-500' : 'border-gray-300'
          }`}
        />
        {errors.password && <ErrorMessage message={errors.password} />}
      </div>
      <div className='relative my-2 w-full'>
        <label
          htmlFor='confirm-password'
          className='text-2xs font-medium text-gray-700'
        >
          Confirm New Password
        </label>
        <input
          required
          name='confirm-password'
          type='password'
          value={confirmPassword}
          onChange={e => {
            setConfirmPassword(e.target.value)
            setErrors({})
            setError('')
          }}
          className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
            errors.confirmPassword ? 'border-red-500' : 'border-gray-300'
          }`}
        />
        {errors.confirmPassword && (
          <ErrorMessage message={errors.confirmPassword} />
        )}
      </div>
      <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
        <button
          type='submit'
          disabled={submitting}
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-black focus:ring-offset-2'
        >
          Reset Password
        </button>
      </div>
      {error && <ErrorMessage message={error} />}
    </form>
  )
}

export default function Account() {
  const { data: auth } = useSWR('/api/users/self')

  const [showNotification, setshowNotification] = useState(false)

  const hasInfraProvider = auth?.providerNames.includes('infra')

  return (
    <>
      <Head>
        <title>Account - Infra</title>
      </Head>
      <h1 className='my-6 py-1 text-lg font-medium'>Account settings</h1>
      {auth && hasInfraProvider && (
        <div className='flex flex-1 flex-col'>
          <h2 className='text-md py-2 font-medium text-gray-600'>
            Reset Password
          </h2>
          <div className='flex max-w-md flex-col space-y-2'>
            <PasswordReset
              onReset={() => {
                setshowNotification(true)
                setTimeout(() => {
                  setshowNotification(false)
                }, 5000)
              }}
            />
          </div>
        </div>
      )}
      {/* Notification */}
      <Notification
        show={showNotification}
        setShow={setshowNotification}
        text='Password Successfully Reset'
      />
    </>
  )
}

Account.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
