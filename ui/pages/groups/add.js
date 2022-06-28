import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'

import ErrorMessage from '../../components/error-message'
import Fullscreen from '../../components/layouts/fullscreen'

export default function GroupsAdd() {
  const [groupName, setGroupName] = useState('')
  const [emails, setEmails] = useState([])
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  const handleGroupNameInputChange = value => {
    setGroupName(value)
    setError('')
  }

  const handleAddGroup = () => {}

  return (
    <>
      <Head>Create Group</Head>
      <div className='space-y-4 pt-5 pb-4'>
        <div className='flex flex-col'>
          <div className='flex flex-row items-center space-x-2 px-4'>
            <img alt='groups' src='/groups.svg' className='h-6 w-6' />
            <div>
              <h1 className='text-2xs'>Create Group</h1>
            </div>
          </div>
          <div className='mt-6 flex flex-col space-y-1'>
            <div className='mt-4 px-4'>
              <label className='text-3xs uppercase text-gray-400'>
                Name Your Group
              </label>
              <input
                autoFocus
                spellCheck='false'
                type='search'
                placeholder='enter the group name here'
                value={groupName}
                onChange={e => handleGroupNameInputChange(e.target.value)}
                className={`border-gray-950 w-full border-b bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
                  errors ? 'border-pink-500' : 'border-gray-800'
                }`}
              />
              {errors && <ErrorMessage message={errors.name} />}
            </div>
            <section className='flex flex-col pt-10 pb-2'>
              <input className='min-h-[120px] bg-gray-900 text-2xs text-gray-300' />
              {/* {submitted ? command : ''} */}
            </section>
          </div>
          <div className='mt-6 flex flex-row items-center justify-end'>
            <Link href='/groups'>
              <a className='-ml-4 border-0 px-4 py-2 text-4xs uppercase text-gray-400 hover:text-white'>
                Cancel
              </a>
            </Link>
            <button
              type='button'
              onClick={() => handleAddGroup()}
              disabled={!groupName || emails.length === 0}
              className='flex-none self-end rounded-md border border-violet-300 px-4 py-2 text-2xs text-violet-100 disabled:opacity-10'
            >
              Create User
            </button>
          </div>
        </div>
        {error && <ErrorMessage message={error} />}
      </div>
    </>
  )
}

GroupsAdd.layout = page => <Fullscreen closeHref='/groups'>{page}</Fullscreen>
