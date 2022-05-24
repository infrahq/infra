import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'
import ErrorMessage from '../../components/error-message'

import Fullscreen from '../../components/layouts/fullscreen'
import { validateEmail } from '../../lib/email'

function AddUser ({ email, handleOnChange, handleKeyDown, handleAddUserOnClick, error }) {
  return (
    <div className='flex flex-col pt-8 px-1 border rounded-lg border-gray-950'>
      <div className='flex flex-row space-x-2 items-center px-4'>
        <img src='/users.svg' className='w-6 h-6' />
        <div>
          <h1 className='text-2xs'>Add User</h1>
        </div>
      </div>
      <div className='flex flex-col mt-6 mx-4 space-y-1'>
        <div className='mt-4'>
          <label className='text-3xs text-gray-300 uppercase'>User Email</label>
          <input
            autoFocus
            spellCheck='false'
            type='email'
            placeholder='enter the user email here'
            value={email}
            onChange={handleOnChange}
            onKeyDown={handleKeyDown}
            className={`w-full bg-transparent border-b border-gray-950 text-3xs px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${error ? 'border-pink-500' : 'border-gray-800'}`}
          />
        </div>
        {error && <ErrorMessage message={error} />}
      </div>
      <div className='flex flex-row justify-between m-6 items-center'>
        <Link href='/users'>
          <a className='uppercase border-0 hover:text-white text-gray-300 text-4xs'>Cancel</a>
        </Link>
        <button
          type='button'
          onClick={handleAddUserOnClick}
          disabled={!email}
          className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 text-center disabled:opacity-30'
        >
          <div className='bg-black rounded-md tracking-tight text-2xs px-6 py-3 text-pink-200'>
            Add User
          </div>
        </button>
      </div>
    </div>
  )
}

function UserOneTimePassword ({ password, handleAddUserOnClick }) {
  return (
    <div className='flex flex-col pt-8 px-1 border rounded-lg border-gray-950'>
      <div className='flex flex-row space-x-2 items-center px-4'>
        <img src='/users.svg' className='w-6 h-6' />
        <div>
          <h1 className='text-2xs'>Add User</h1>
        </div>
      </div>
      <h2 className='text-2xs pt-5 px-4'>User added. Send the user this one time password for their initial login. This password will not be shown again.</h2>
      <div className='flex flex-col mt-6 mx-4 space-y-1'>
        <label className='text-3xs text-gray-300 uppercase'>One Time Password</label>
        <input
          readOnly
          value={password}
          className='w-full bg-transparent text-3xs px-px py-3 focus:outline-none font-mono'
        />
      </div>
      <div className='flex flex-row justify-between m-6 items-center'>
        <button onClick={handleAddUserOnClick} className='uppercase border-0 hover:text-white text-gray-300 text-4xs'>Add Another</button>
        <Link href='/users'>
          <a className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 text-center disabled:opacity-30'>
            <div className='bg-black rounded-md tracking-tight text-2xs px-6 py-3 text-pink-200'>
              Done
            </div>
          </a>
        </Link>
      </div>
    </div>
  )
}

export default function () {
  const [email, setEmail] = useState('')
  const [state, setState] = useState('add')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  const handleGetOneTimePassword = async () => {
    if (validateEmail(email)) {
      setErrors({})
      try {
        const res = await fetch('/v1/identities', {
          method: 'POST',
          body: JSON.stringify({ name: email, setOneTimePassword: true })
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
            errors[error.fieldName.toLowerCase()] = error.errors[0] || 'invalid value'
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

  const handleInputChang = value => {
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
    <Fullscreen closeHref='/users'>
      <Head>
        <title>Add User</title>
      </Head>
      <div className='w-full max-w-sm'>
        {state === 'add' && <AddUser email={email} handleOnChange={e => handleInputChang(e.target.value)} handleKeyDown={e => handleKeyDownEvent(e.key)} handleAddUserOnClick={() => handleGetOneTimePassword()} error={errors.name} />}
        {state === 'password' && <UserOneTimePassword password={password} handleAddUserOnClick={() => handleAddUser()} />}
        {error && <ErrorMessage message={error} />}
      </div>
    </Fullscreen>
  )
}
