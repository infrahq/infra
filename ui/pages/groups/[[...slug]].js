import Head from 'next/head'
import { useRouter } from 'next/router'
import useSWR from 'swr'
import { useState, useRef } from 'react'
import dayjs from 'dayjs'
import { Combobox } from '@headlessui/react'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import PageHeader from '../../components/page-header'
import EmptyTable from '../../components/empty-table'
import Table from '../../components/table'
import Sidebar from '../../components/sidebar'
import DeleteModal from '../../components/delete-modal'
import EmailBadge from '../../components/email-badge'
import { PlusIcon } from '@heroicons/react/outline'

const columns = [
  {
    Header: 'Name',
    accessor: g => g,
    width: '80%',
    Cell: ({ value: group }) => {
      return (
        <div className='flex items-center py-2'>
          <div className='flex h-6 w-6 select-none items-center justify-center rounded-md border border-violet-300/40'>
            <img alt='group icon' src='/groups.svg' className='h-3 w-3' />
          </div>
          <div className='ml-3 flex min-w-0 flex-1 flex-col leading-tight'>
            <div className='truncate'>{group.name}</div>
          </div>
        </div>
      )
    },
  },
  {
    Header: 'Team Size',
    accessor: g => g,
    width: '20%',
    Cell: ({ value: group }) => {
      const { data: { items: users } = {}, error } = useSWR(
        `/api/users?group=${group.id}`
      )

      return (
        <>
          {users && (
            <div className='text-gray-400'>
              {users?.length} {users?.length > 1 ? 'members' : 'member'}
            </div>
          )}
          {error?.status && <div className='text-gray-400'>--</div>}
        </>
      )
    },
  },
]

// TODO: refactor
function EmailsSelectInput({
  selectedEmails,
  setSelectedEmails,
  existMembers,
  onClick,
}) {
  const { data: { items: users } = { items: [] } } = useSWR('/api/users')

  console.log(existMembers)

  const [query, setQuery] = useState('')
  const button = useRef()
  const inputRef = useRef(null)

  const selectedEmailsId = selectedEmails.map(i => i.id)

  const filteredEmail = [...users.map(u => ({ ...u, user: true }))]
    .filter(s => s?.name?.toLowerCase()?.includes(query.toLowerCase()))
    .filter(s => s.name !== 'connector')
    .filter(s => !selectedEmailsId?.includes(s.id))
    .filter(s => !existMembers?.includes(s.id))

  const removeSelectedEmail = email => {
    setSelectedEmails(selectedEmails.filter(item => item.id !== email.id))
  }

  const handleKeyDownEvent = key => {
    if (key === 'Backspace' && inputRef.current.value.length === 0) {
      removeSelectedEmail(selectedEmails[selectedEmails.length - 1])
    }
  }

  return (
    <section className='my-2 flex'>
      <div className='flex flex-1 items-center border-b border-gray-800 py-3'>
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
                placeholder={
                  selectedEmails.length === 0 ? 'Add email here' : ''
                }
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
      <div className='relative'>
        <button
          type='button'
          onClick={onClick}
          disabled={selectedEmails.length === 0}
          className='flex h-10 cursor-pointer items-center rounded-md border border-violet-300 px-3 py-3 text-2xs disabled:transform-none disabled:cursor-default disabled:opacity-30 disabled:transition-none sm:ml-4 sm:mt-0'
        >
          <PlusIcon className='mr-1.5 h-3 w-3' />
          <div className='text-violet-100'>Add</div>
        </button>
      </div>
    </section>
  )
}

function Details({ group, admin }) {
  const { id, name } = group

  const { data: { items: users } = {}, mutate } = useSWR(
    `/api/users?group=${group.id}`
  )

  console.log(users)

  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [emails, setEmails] = useState([])

  const existMembers = users?.map(m => m.id)

  return (
    <div className='flex flex-1 flex-col space-y-6'>
      {admin && (
        <>
          <section>
            <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
              Access
            </h3>
          </section>
          <section>
            <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
              Team
            </h3>
            <EmailsSelectInput
              selectedEmails={emails}
              setSelectedEmails={setEmails}
              existMembers={existMembers}
              onClick={async () => {
                const usersToAdd = emails.map(email => email.id)
                await fetch(`/api/groups/${id}/users`, {
                  method: 'PATCH',
                  body: JSON.stringify({ usersToAdd }),
                })

                await mutate(`/api/users?group=${group.id}`)
                setEmails([])
              }}
            />
            <div className='mt-4'>
              {users?.length === 0 && (
                <div className='mt-6 text-2xs italic text-gray-400'>
                  No members in the group
                </div>
              )}

              {users
                ?.sort((a, b) => b.created?.localeCompare(a.created))
                .map(u => (
                  <div
                    key={u.id}
                    className='group flex items-center justify-between truncate text-2xs'
                  >
                    <div className='py-2'>{u.name}</div>
                    <div className='flex justify-end text-right opacity-0 group-hover:opacity-100'>
                      <button
                        onClick={async () => {
                          const usersToRemove = [u.id]
                          await fetch(`/api/groups/${id}/users`, {
                            method: 'PATCH',
                            body: JSON.stringify({ usersToRemove }),
                          })
                          mutate({ items: users.filter(i => i.id !== u.id) })
                        }}
                        className='-mr-2 flex-none cursor-pointer px-2 py-1 text-2xs text-gray-500 hover:text-violet-100'
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ))}
            </div>
          </section>
        </>
      )}
      <section>
        <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
          Metadata
        </h3>
        <div className='flex flex-col space-y-2 pt-3'>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>ID</div>
            <div className='text-2xs'>{id}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Created</div>
            <div className='text-2xs'>
              {group?.created ? dayjs(group.created).fromNow() : '-'}
            </div>
          </div>
        </div>
      </section>
      {admin && (
        <section className='flex flex-1 flex-col items-end justify-end py-6'>
          <button
            type='button'
            onClick={() => setDeleteModalOpen(true)}
            className='flex items-center rounded-md border border-violet-300 px-6 py-3 text-2xs text-violet-100'
          >
            Remove
          </button>
          <DeleteModal
            open={deleteModalOpen}
            setOpen={setDeleteModalOpen}
            onSubmit={() => {}}
            title='Remove Group'
            message={
              <>
                Are you sure you want to delete{' '}
                <span className='font-bold text-white'>{name}</span>? This
                action cannot be undone.
              </>
            }
          />
        </section>
      )}
    </div>
  )
}

export default function Groups() {
  const { data: { items: groups } = {}, error } = useSWR('/api/groups')
  const { admin, loading: adminLoading } = useAdmin()
  const router = useRouter()

  const loading = adminLoading || (!groups && !error)
  const { slug: [id] = [] } = router.query

  const group = groups?.find(g => g.id === id)

  if (id && groups && !group) {
    router.replace('/groups')
    return null
  }

  return (
    <>
      <Head>Groups - Infra</Head>
      {!loading && (
        <div className='flex h-full flex-1'>
          <div className='flex flex-1 flex-col space-y-4'>
            <PageHeader
              header='Groups'
              buttonHref='/groups/add'
              buttonLabel='Group'
            />
            {error?.status ? (
              <div className='my-20 text-center text-sm font-light text-gray-300'>
                {error?.info?.message}
              </div>
            ) : (
              <div className='flex min-h-0 flex-1 flex-col overflow-y-scroll px-6'>
                <Table
                  columns={columns}
                  data={
                    groups?.sort((a, b) =>
                      b.created?.localeCompare(a.created)
                    ) || []
                  }
                  getRowProps={row => ({
                    onClick: () => router.push(`/groups/${row.original.id}`),
                    className:
                      id === row.original.id
                        ? 'bg-gray-900/50'
                        : 'cursor-pointer',
                  })}
                />
                {groups?.length === 0 && (
                  <EmptyTable
                    title='There are no groups'
                    subtitle='Connect, create and manage your groups.'
                    iconPath='/groups.svg'
                    buttonHref='/groups/add'
                    buttonText='Groups'
                  />
                )}
              </div>
            )}
          </div>
          {id && (
            <Sidebar
              handleClose={() => router.push('/groups')}
              title={group?.name}
              iconPath='/groups.svg'
            >
              <Details group={groups?.find(g => g.id === id)} admin={admin} />
            </Sidebar>
          )}
        </div>
      )}
    </>
  )
}

Groups.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
