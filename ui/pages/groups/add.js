import Head from 'next/head'
import Link from 'next/link'
import { useState, useRef } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'

import ErrorMessage from '../../components/error-message'
import Fullscreen from '../../components/layouts/fullscreen'
import TypeaheadCombobox from '../../components/typeahead-combobox'

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
    <div className='bg-gray-900 px-4 py-3'>
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
                  errors.name ? 'border-pink-500' : 'border-gray-800'
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
          <div className='mt-6 flex flex-row items-center justify-end px-4'>
            <Link href='/groups'>
              <a className='-ml-4 border-0 px-4 py-2 text-4xs uppercase text-gray-400 hover:text-white'>
                Cancel
              </a>
            </Link>
            <button
              type='button'
              onClick={() => handleCreateGroup()}
              disabled={!groupName}
              className='flex-none self-end rounded-md border border-violet-300 px-4 py-2 text-2xs text-violet-100 disabled:opacity-10'
            >
              Create Group
            </button>
          </div>
        </div>
        {error && <ErrorMessage message={error} />}
      </div>
    </>
  )
}

GroupsAdd.layout = page => <Fullscreen closeHref='/groups'>{page}</Fullscreen>
