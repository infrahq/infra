import Head from 'next/head'
import { useState, Fragment } from 'react'
import useSWR from 'swr'
import { CheckCircleIcon } from '@heroicons/react/outline'
import { XIcon } from '@heroicons/react/solid'
import { Transition } from '@headlessui/react'

import Dashboard from '../../components/layouts/dashboard'

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
    <form onSubmit={onSubmit} className='flex max-w-md flex-col'>
      <div className='relative my-2 w-full'>
        <label htmlFor='password' className='text-sm font-medium'>
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
        {errors.password && (
          <p className='absolute text-xs text-red-500'>{errors.password}</p>
        )}
      </div>
      <div className='relative my-2 w-full'>
        <label htmlFor='confirm-password' className='text-sm font-medium'>
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
          <p className='absolute text-xs text-red-500'>
            {errors.confirmPassword}
          </p>
        )}
      </div>
      <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
        <button
          type='submit'
          disabled={submitting}
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-black focus:ring-offset-2'
        >
          Reset Password
        </button>
      </div>
      {error && <p className='text-xs text-red-500'>{error}</p>}
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
      <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
        <div className='px-4 sm:px-6 xl:px-0'>
          {auth && hasInfraProvider && (
            <div className='flex flex-1 flex-col space-y-8'>
              <div className='py-12'>
                <h1 className='text-lg font-medium'>Reset Password</h1>
                <div className='flex flex-col space-y-2 pt-6'>
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

              {/* Notification */}
              <div
                aria-live='assertive'
                className='pointer-events-none fixed inset-0 flex items-end px-4 py-6 sm:items-end sm:p-6'
              >
                <div className='flex w-full flex-col items-center space-y-4 sm:items-end'>
                  <Transition
                    show={showNotification}
                    as={Fragment}
                    enter='transform ease-out duration-300 transition'
                    enterFrom='translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2'
                    enterTo='translate-y-0 opacity-100 sm:translate-x-0'
                    leave='transition ease-in duration-100'
                    leaveFrom='opacity-100'
                    leaveTo='opacity-0'
                  >
                    <div className='pointer-events-auto w-full max-w-sm overflow-hidden rounded-lg bg-white shadow-lg ring-1 ring-black ring-opacity-5'>
                      <div className='p-4'>
                        <div className='flex items-start'>
                          <div className='flex-shrink-0'>
                            <CheckCircleIcon
                              className='h-6 w-6 text-green-400'
                              aria-hidden='true'
                            />
                          </div>
                          <div className='ml-3 w-0 flex-1 pt-0.5'>
                            <p className='text-sm font-medium text-gray-900'>
                              Password Successfully Reset
                            </p>
                          </div>
                          <div className='ml-4 flex flex-shrink-0'>
                            <button
                              type='button'
                              className='inline-flex rounded-md bg-white text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2'
                              onClick={() => setshowNotification(false)}
                            >
                              <span className='sr-only'>Close</span>
                              <XIcon className='h-5 w-5' aria-hidden='true' />
                            </button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </Transition>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </>
  )
}

Account.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
