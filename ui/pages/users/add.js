import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'

import Fullscreen from '../../components/modals/fullscreen'

export default function () {
  const [email, setEmail] = useState('')
  const [state, setState] = useState('add')
  const [error, setError] = useState('')

  return (
    <Fullscreen closeHref='/users'>
      <Head>
        <title>Add User</title>
      </Head>
      <div className='w-full max-w-sm'>
        <div className='flex flex-col pt-8 px-1 border rounded-lg border-gray-950'>
          <div className='flex flex-row space-x-2 items-center px-4'>
            <img src='/users.svg' className='w-6 h-6' />
            <div>
              <h1 className='text-name'>Add User</h1>
            </div>
          </div>
          <div className='flex flex-col mt-11 mx-6 space-y-1'>
            <div className='mt-4'>
              <label className='text-label text-gray-300 uppercase'>User Email</label>
              <input
                autoFocus
                spellCheck='false'
                type='email'
                autocomplete='off'
                placeholder='enter the user email here'
                value={email}
                onChange={e => setEmail(e.target.value)}
                className='w-full bg-transparent border-b border-gray-950 text-label px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic'
              />
            </div>
          </div>
          <div className='flex flex-row justify-between m-6 items-center'>
            <Link href='/users'>
              <a className='uppercase border-0 hover:text-white text-gray-300 text-secondary'>Cancel</a>
            </Link>
            <button
              type='button'
              className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 text-center disabled:opacity-30'
            >
              <div className='bg-black rounded-md tracking-tight text-name px-6 py-3 text-pink-200'>
                Add User
              </div>
            </button>
          </div>
        </div>
      </div>
    </Fullscreen>
  )
}
