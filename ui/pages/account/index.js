import Head from 'next/head'
import { useState, useEffect, useRef } from 'react'

import { useUser } from '../../lib/hooks'

import Dashboard from '../../components/layouts/dashboard'
import Notification from '../../components/notification'

function PasswordInput({ title, name, value, onChange, error }) {
  return (
    <div>
      <label htmlFor={name} className='text-2xs font-medium text-gray-700'>
        {title}
      </label>
      <input
        required
        name={name}
        type='password'
        autoComplete='off'
        value={value}
        onChange={e => onChange(e)}
        className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
          error ? 'border-red-500' : 'border-gray-300'
        }`}
      />
      {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
    </div>
  )
}

function PasswordReset({ user, onReset = () => {} }) {
  const [oldPassword, setOldPassword] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  async function onSubmit(e) {
    e.preventDefault()

    if (password !== confirmPassword) {
      setErrors({
        confirmPassword: 'passwords do not match',
      })
      return false
    }

    setError('')
    setErrors({})

    try {
      const rest = await fetch(`/api/users/${user?.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          ...user,
          oldPassword,
          password: confirmPassword,
        }),
      })

      const data = await rest.json()

      if (!rest.ok) {
        throw data
      }

      setOldPassword('')
      setPassword('')
      setConfirmPassword('')
      onReset()
    } catch (e) {
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
      <div className='relative w-full space-y-2'>
        <PasswordInput
          name='old-password'
          title='Old Password'
          onChange={e => {
            setOldPassword(e.target.value)
            setErrors({})
            setError('')
          }}
          value={oldPassword}
          error={errors.oldpassword}
        />
        <PasswordInput
          name='password'
          title='New Password'
          onChange={e => {
            setPassword(e.target.value)
            setErrors({})
            setError('')
          }}
          value={password}
          error={errors.password}
        />
        <PasswordInput
          name='confirm-password'
          title='Confirm New Password'
          onChange={e => {
            setConfirmPassword(e.target.value)
            setErrors({})
            setError('')
          }}
          value={confirmPassword}
          error={errors.confirmPassword}
        />
      </div>
      <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
        <button
          type='submit'
          disabled={
            !(
              oldPassword &&
              password &&
              confirmPassword &&
              Object.keys(errors).length === 0 &&
              error === ''
            )
          }
          className='inline-flex cursor-pointer items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-black focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-30'
        >
          Reset Password
        </button>
      </div>
      {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
    </form>
  )
}

export default function Account() {
  const { user } = useUser()

  const [showNotification, setShowNotification] = useState(false)

  const timerRef = useRef(null)

  const hasInfraProvider = user?.providerNames?.includes('infra')

  useEffect(() => {
    return clearTimer()
  }, [])

  function clearTimer() {
    setShowNotification(false)
    return clearTimeout(timerRef.current)
  }

  return (
    <div className='mx-auto w-full max-w-2xl'>
      <Head>
        <title>Account - Infra</title>
      </Head>
      <h1 className='my-6 py-1 font-display text-lg font-medium'>
        Account settings
      </h1>
      {user && hasInfraProvider && (
        <div className='flex flex-1 flex-col'>
          <h2 className='text-md py-2 font-medium text-gray-600'>
            Reset Password
          </h2>
          <div className='flex flex-col space-y-2'>
            <PasswordReset
              user={user}
              onReset={() => {
                setShowNotification(true)
                setTimeout(() => {
                  setShowNotification(false)
                }, 5000)
              }}
            />
          </div>
        </div>
      )}
      {/* Notification */}
      <Notification
        show={showNotification}
        setShow={setShowNotification}
        text='Password Successfully Reset'
        setClearNotification={() => clearTimer()}
      />
    </div>
  )
}

Account.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
