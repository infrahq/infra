import { UserIcon } from '@heroicons/react/outline'
import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'
import { useServerConfig } from '../../lib/serverconfig'

import ErrorMessage from '../../components/error-message'
import Dashboard from '../../components/layouts/dashboard'

function AddUser({ email, onChange, onKeyDown, onSubmit, error }) {
  return (
    <form onSubmit={onSubmit} className='flex flex-col'>
      <div className='mt-6 flex flex-col space-y-1'>
        <div className='mt-4'>
          <label className='text-2xs font-medium text-gray-700'>
            User Email
          </label>
          <input
            autoFocus
            spellCheck='false'
            type='email'
            value={email}
            onChange={onChange}
            onKeyDown={onKeyDown}
            className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
              error ? 'border-red-500' : 'border-gray-300'
            }`}
          />
        </div>
        {error && <ErrorMessage message={error} />}
      </div>
      <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
        <button
          type='submit'
          disabled={!email}
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'
        >
          Add User
        </button>
      </div>
    </form>
  )
}

function UserOneTimePassword({ isEmailConfigured, password, onSubmit }) {
  return (
    <div className='flex flex-col'>
      {isEmailConfigured ? (
        <h2 className='mt-5 text-sm'>
          User added. The user has been emailed a link inviting them to join.
        </h2>
      ) : (
        <div>
          <h2 className='mt-5 text-sm'>
            User added. Send the user this temporary password for their initial
            login. This password will not be shown again.
          </h2>
          <div className='mt-6 flex flex-col space-y-3'>
            <label className='text-2xs font-medium text-gray-700'>
              Temporary Password
            </label>
            <input
              readOnly
              value={password}
              className='round-md my-0 w-full bg-gray-100 p-2 font-mono text-xs focus:outline-none'
            />
          </div>
        </div>
      )}

      <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
        <button
          onClick={onSubmit}
          className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-2xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
        >
          Add Another
        </button>
        <Link href='/users'>
          <a className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'>
            Done
          </a>
        </Link>
      </div>
    </div>
  )
}

export default function UsersAdd() {
  const [email, setEmail] = useState('')
  const [state, setState] = useState('add')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const { isEmailConfigured } = useServerConfig()

  async function handleUserOneTimePassword(e) {
    e.preventDefault()

    setErrors({})
    setError('')

    try {
      const res = await fetch('/api/users', {
        method: 'POST',
        body: JSON.stringify({
          name: email,
        }),
      })
      const user = await res.json()

      if (!res.ok) {
        throw user
      }

      setState('password')
      setPassword(user.oneTimePassword)
    } catch (e) {
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] =
            error.errors[0] || 'invalid value'
        }

        // TODO: need to work with backend for better error message
        if (e.code === 409 && errors.identity_id) {
          errors.name = 'user already exists'
        }

        setErrors(errors)
      } else {
        setError(e.message)
      }

      return false
    }
  }

  function handleInputChange(value) {
    setEmail(value)
    setError('')
  }

  function handleAddUser() {
    setState('add')
    setEmail('')
    setPassword('')
  }

  function handleKeyDownEvent(e) {
    if (e.key === 'Enter' && email.length > 0) {
      handleUserOneTimePassword(e)
    }
  }

  return (
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      <Head>
        <title>Add User</title>
      </Head>
      <div className='space-y-4 px-4 py-5 md:px-6 xl:px-0'>
        {state === 'add' && (
          <AddUser
            email={email}
            onChange={e => {
              handleInputChange(e.target.value)
              setErrors({})
              setError('')
            }}
            onKeyDown={e => handleKeyDownEvent(e)}
            onSubmit={handleUserOneTimePassword}
            error={errors.name}
          />
        )}
        {state === 'password' && (
          <UserOneTimePassword
            isEmailConfigured={isEmailConfigured}
            password={password}
            onSubmit={() => handleAddUser()}
          />
        )}
        {error && <ErrorMessage message={error} />}
      </div>
    </div>
  )
}

UsersAdd.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
