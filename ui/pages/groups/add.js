import Head from 'next/head'
import Link from 'next/link'
import { useState, useRef } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'
import { UserGroupIcon } from '@heroicons/react/outline'

import ErrorMessage from '../../components/error-message'
import TypeaheadCombobox from '../../components/typeahead-combobox'
import Dashboard from '../../components/layouts/dashboard'

function EmailsSelectInput({ selectedEmails, setSelectedEmails }) {
  const { data: { items: users } = { items: [] } } = useSWR(
    '/api/users?limit=1000'
  )

  const [query, setQuery] = useState('')
  const inputRef = useRef(null)

  const selectedEmailsId = selectedEmails.map(i => i.id)

  const filteredEmail = [...users.map(u => ({ ...u, user: true }))]
    .filter(s => s?.name?.toLowerCase()?.includes(query.toLowerCase()))
    .filter(s => !selectedEmailsId?.includes(s.id))

  function removeSelectedEmail(email) {
    setSelectedEmails(selectedEmails.filter(item => item.id !== email.id))
  }

  function handleKeyDownEvent(key) {
    if (key === 'Backspace' && inputRef.current.value.length === 0) {
      removeSelectedEmail(selectedEmails[selectedEmails.length - 1])
    }
  }

  return (
    <div className='rounded-md bg-gray-100 p-2'>
      <TypeaheadCombobox
        selectedEmails={selectedEmails}
        setSelectedEmails={setSelectedEmails}
        onRemove={removedEmail => removeSelectedEmail(removedEmail)}
        inputRef={inputRef}
        setQuery={setQuery}
        filteredEmail={filteredEmail}
        onKeyDownEvent={key => handleKeyDownEvent(key)}
      />
    </div>
  )
}

export default function GroupsAdd() {
  const { mutate } = useSWRConfig()

  const [groupName, setGroupName] = useState('')
  const [emails, setEmails] = useState([])
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  const router = useRouter()

  const handleGroupNameInputChange = value => {
    setGroupName(value)
    setError('')
  }

  const addUsersToGroup = async groupId => {
    const usersToAdd = emails.map(email => email.id)

    try {
      const res = await fetch(`/api/groups/${groupId}/users`, {
        method: 'PATCH',
        body: JSON.stringify({ usersToAdd }),
      })

      const data = await res.json()

      if (!res.ok) {
        throw data
      }

      await mutate('/api/groups')

      router.replace('/groups')
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

  const handleCreateGroup = async () => {
    setErrors({})
    setError('')

    try {
      const res = await fetch('/api/groups', {
        method: 'POST',
        body: JSON.stringify({ name: groupName }),
      })

      const group = await res.json()

      if (!res.ok) {
        throw group
      } else {
        addUsersToGroup(group.id)
      }
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
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      <Head>
        <title>Create Group</title>
      </Head>
      <div className='space-y-4 px-4 py-5 md:px-6 xl:px-0'>
        <div className='flex flex-col'>
          <div className='flex flex-row items-center space-x-2'>
            <UserGroupIcon className='h-6 w-6' />
            <div>
              <h1 className='text-base'>Create Group</h1>
            </div>
          </div>
          <div className='mt-6 flex flex-col space-y-1'>
            <div className='mt-4'>
              <label className='text-2xs font-medium text-gray-700'>
                Name Your Group
              </label>
              <input
                autoFocus
                spellCheck='false'
                type='search'
                value={groupName}
                onChange={e => handleGroupNameInputChange(e.target.value)}
                className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                  errors.name ? 'border-red-500' : 'border-gray-300'
                }`}
              />
              {errors && <ErrorMessage message={errors.name} />}
            </div>
            <section className='flex flex-col pt-10 pb-2'>
              <EmailsSelectInput
                selectedEmails={emails}
                setSelectedEmails={setEmails}
              />
            </section>
          </div>
          <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
            <button
              type='button'
              onClick={() => handleCreateGroup()}
              disabled={!groupName}
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'
            >
              Create Group
            </button>
          </div>
        </div>
        {error && <ErrorMessage message={error} />}
      </div>
    </div>
  )
}

GroupsAdd.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
