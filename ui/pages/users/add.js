import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'
import ErrorMessage from '../../components/error-message'

import Fullscreen from '../../components/layouts/fullscreen'
import { validateEmail } from '../../lib/email'

function AddUser({ email, onChange, onKeyDown, onAddUser, error }) {
  return (
    <div className='flex flex-col'>
      <div className='flex flex-row items-center space-x-2'>
        <img alt='users' src='/users.svg' className='h-6 w-6' />
        <div>
          <h1 className='text-2xs'>Add User</h1>
        </div>
      </div>
      <div className='mt-6 flex flex-col space-y-1'>
        <div className='mt-4'>
          <label className='text-3xs uppercase text-gray-400'>User Email</label>
          <input
            autoFocus
            spellCheck='false'
            type='email'
            placeholder='enter the user email here'
            value={email}
            onChange={onChange}
            onKeyDown={onKeyDown}
            className={`border-gray-950 w-full border-b bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              error ? 'border-pink-500' : 'border-gray-800'
            }`}
          />
        </div>
        {error && <ErrorMessage message={error} />}
      </div>
      <div className='mt-6 flex flex-row items-center justify-end'>
        <Link href='/users'>
          <a className='-ml-4 border-0 px-4 py-2 text-4xs uppercase text-gray-400 hover:text-white'>
            Cancel
          </a>
        </Link>
        <button
          type='button'
          onClick={onAddUser}
          disabled={!email}
          className='flex-none self-end rounded-md border border-violet-300 px-4 py-2 text-2xs text-violet-100 disabled:opacity-10'
        >
          Add User
        </button>
      </div>
    </div>
  )
}

function UserOneTimePassword({ password, onAddUser }) {
  return (
    <div className='flex flex-col'>
      <div className='flex flex-row items-center space-x-2'>
        <img alt='users icon' src='/users.svg' className='h-6 w-6' />
        <div>
          <h1 className='text-2xs'>Add User</h1>
        </div>
      </div>
      <h2 className='mt-5 text-2xs'>
        User added. Send the user this temporary password for their initial
        login. This password will not be shown again.
      </h2>
      <div className='mt-6 flex flex-col space-y-1'>
        <label className='text-3xs uppercase text-gray-400'>
          Temporary Password
        </label>
        <input
          readOnly
          value={password}
          className='my-0 w-full bg-transparent py-2 font-mono text-3xs focus:outline-none'
        />
      </div>
      <div className='mt-6 flex flex-row items-center justify-end'>
        <button
          onClick={onAddUser}
          className='border-0 px-4 py-2 text-4xs uppercase text-gray-400 hover:text-white'
        >
          Add Another
        </button>
        <Link href='/users'>
          <a className='flex-none self-end rounded-md border border-violet-300 px-8 py-2 text-2xs text-violet-100 disabled:opacity-10'>
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

  const handleGetOneTimePassword = async () => {
    if (validateEmail(email)) {
      setErrors({})
      try {
        const res = await fetch('/api/users', {
          method: 'POST',
          body: JSON.stringify({ name: email, setOneTimePassword: true }),
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
          setErrors(errors)
        } else {
          setError(e.message)
        }

        return false
      }
    } else {
      setErrors({ name: 'Invalid email' })
    }
  }

  const handleInputChange = value => {
    setEmail(value)
    setError('')
  }

  const handleAddUser = () => {
    setState('add')
    setEmail('')
    setPassword('')
  }

  const handleKeyDownEvent = key => {
    if (key === 'Enter' && email.length > 0) {
      handleGetOneTimePassword()
    }
  }

  return (
    <>
      <Head>
        <title>Add User</title>
      </Head>
      <div className='space-y-4 px-4 pt-5 pb-4'>
        {state === 'add' && (
          <AddUser
            email={email}
            onChange={e => handleInputChange(e.target.value)}
            onKeyDown={e => handleKeyDownEvent(e.key)}
            onAddUser={() => handleGetOneTimePassword()}
            error={errors.name}
          />
        )}
        {state === 'password' && (
          <UserOneTimePassword
            password={password}
            onAddUser={() => handleAddUser()}
          />
        )}
        {error && <ErrorMessage message={error} />}
      </div>
    </>
  )
}

UsersAdd.layout = page => <Fullscreen closeHref='/users'>{page}</Fullscreen>
