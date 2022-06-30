import { Combobox } from '@headlessui/react'
import Head from 'next/head'
import Link from 'next/link'
import { useState, useRef } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import { XIcon } from '@heroicons/react/outline'
import { useRouter } from 'next/router'

import ErrorMessage from '../../components/error-message'
import Fullscreen from '../../components/layouts/fullscreen'

function EmailBadge({ email, onRemove }) {
  return (
    <div className='m-1 flex items-center justify-center overflow-hidden text-ellipsis rounded-md bg-gray-800 py-1 px-2 font-medium text-white'>
      <div className='max-w-full flex-initial overflow-hidden text-ellipsis text-xs font-normal leading-none'>
        {email}
      </div>
      <div className='flex flex-auto flex-row-reverse pl-1'>
        <XIcon
          className='h-2 w-2 hover:cursor-pointer'
          aria-hidden='true'
          onClick={onRemove}
        />
      </div>
    </div>
  )
}

function EmailsSelectInput({ selectedEmails, setSelectedEmails }) {
  const { data: { items: users } = { items: [] } } = useSWR('/api/users')

  const [query, setQuery] = useState('')
  const button = useRef()
  const inputRef = useRef(null)

  const selectedEmailsId = selectedEmails.map(i => i.id)

  const filteredEmail = [...users.map(u => ({ ...u, user: true }))]
    .filter(s => s?.name?.toLowerCase()?.includes(query.toLowerCase()))
    .filter(s => s.name !== 'connector')
    .filter(s => !selectedEmailsId?.includes(s.id))

  const removeSelectedEmail = email => {
    setSelectedEmails(selectedEmails.filter(item => item.id !== email.id))
  }

  const handleKeyDownEvent = key => {
    if (key === 'Backspace' && inputRef.current.value.length === 0) {
      removeSelectedEmail(selectedEmails[selectedEmails.length - 1])
    }
  }

  return (
    <div className='bg-gray-900 px-4 py-3'>
      <Combobox
        as='div'
        className='relative flex-1'
        onChange={e => {
          setSelectedEmails([...selectedEmails, e])
        }}
      >
        <div className='flex flex-auto flex-wrap'>
          {selectedEmails?.map(i => (
            <EmailBadge
              key={i.id}
              email={i.name}
              onRemove={() => removeSelectedEmail(i)}
            />
          ))}
          <div className='flex-1'>
            <Combobox.Input
              ref={inputRef}
              className='relative w-full bg-transparent text-xs text-gray-300 placeholder:italic focus:outline-none'
              onChange={e => setQuery(e.target.value)}
              onFocus={() => {
                button.current?.click()
              }}
              onKeyDown={e => handleKeyDownEvent(e.key)}
              placeholder={selectedEmails.length === 0 ? 'Add email here' : ''}
            />
          </div>
        </div>
        {filteredEmail.length > 0 && (
          <Combobox.Options className='absolute -left-[13px] z-10 mt-1 max-h-60 w-56 overflow-auto rounded-md border border-gray-700 bg-gray-800 py-1 text-2xs ring-1 ring-black ring-opacity-5 focus:outline-none'>
            {filteredEmail?.map(f => (
              <Combobox.Option
                key={f.id}
                value={f}
                className={({ active }) =>
                  `relative cursor-default select-none py-2 px-3 hover:bg-gray-700 ${
                    active ? 'bg-gray-700' : ''
                  }`
                }
              >
                <div className='flex flex-row'>
                  <div className='flex min-w-0 flex-1 flex-col'>
                    <div className='flex justify-between py-0.5 font-medium'>
                      <span className='truncate' title={f.name}>
                        {f.name}
                      </span>
                    </div>
                    <div className='text-3xs text-gray-400'>
                      {f.user && 'User'}
                    </div>
                  </div>
                </div>
              </Combobox.Option>
            ))}
          </Combobox.Options>
        )}
        <Combobox.Button className='hidden' ref={button} />
      </Combobox>
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
    const userIDsToAdd = emails.map(email => email.id)

    try {
      const res = await fetch(`/api/groups/${groupId}/users`, {
        method: 'PATCH',
        body: JSON.stringify({ groupID: groupId, userIDsToAdd }),
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
              disabled={!groupName || emails.length === 0}
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
