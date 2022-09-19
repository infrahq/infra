import { useRouter } from 'next/router'
import Link from 'next/link'
import Head from 'next/head'
import useSWR from 'swr'
import { useState, useRef } from 'react'
import { CheckIcon } from '@heroicons/react/outline'
import { Combobox as HeadlessUIComboBox } from '@headlessui/react'

import { useAdmin } from '../../lib/admin'

import Table from '../../components/table'
import RemoveButton from '../../components/remove-button'
import DeleteModal from '../../components/delete-modal'
import Dashboard from '../../components/layouts/dashboard'

function ComboBox({ options = [], selected, setSelected }) {
  const [query, setQuery] = useState('')
  const button = useRef()

  const filtered = options?.filter(u =>
    u.toLowerCase().includes(query.toLowerCase())
  )

  return (
    <HeadlessUIComboBox
      as='div'
      className='w-full'
      value={selected}
      onChange={user => {
        setSelected(user)
      }}
    >
      <div className='relative'>
        <HeadlessUIComboBox.Input
          onFocus={() => {
            if (!selected) {
              button.current?.click()
            }
          }}
          className='w-full rounded-md border border-gray-300 bg-white py-[7px] pl-2.5 pr-10 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 md:text-xs'
          value={query}
          placeholder='User'
          type='search'
          onChange={event => setQuery(event.target.value)}
        />
        <HeadlessUIComboBox.Button className='hidden' ref={button} />

        {filtered?.length > 0 && (
          <HeadlessUIComboBox.Options className='absolute z-10 mt-2 max-h-64 min-w-[16rem] max-w-full overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'>
            {filtered.map(o => (
              <HeadlessUIComboBox.Option
                key={o}
                value={o}
                className={({ active }) =>
                  `relative cursor-default select-none py-2 pl-3 pr-9
                  ${active ? 'bg-gray-100' : 'bg-transparent'}`
                }
              >
                {({ active, selected }) => (
                  <>
                    <span
                      className={`
                          block truncate
                          ${selected ? 'font-medium' : ''}`}
                    >
                      {o}
                    </span>

                    {selected && (
                      <span
                        className={`
                            absolute inset-y-0 right-0 flex items-center pr-4
                            ${active ? 'bg-gray-100' : 'bg-transparent'}`}
                      >
                        <CheckIcon
                          className='h-4 w-4 text-gray-600'
                          aria-hidden='true'
                        />
                      </span>
                    )}
                  </>
                )}
              </HeadlessUIComboBox.Option>
            ))}
          </HeadlessUIComboBox.Options>
        )}
      </div>
    </HeadlessUIComboBox>
  )
}

export default function GroupDetails() {
  const router = useRouter()
  const id = router.query.id
  const page = Math.max(parseInt(router.query.p) || 1, 1)
  const limit = 999
  const { admin } = useAdmin()
  const { data: group, mutate: mutate } = useSWR(`/api/groups/${id}`)
  const {
    data: { items: users, totalCount, totalPages } = {},
    mutate: mutateUsers,
  } = useSWR(`/api/users?group=${group?.id}&limit=${limit}&p=${page}`)
  const { data: { items: allUsers } = {} } = useSWR(`/api/users?limit=999`)
  const { data: { items: infraAdmins } = {} } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=999'
  )
  const [addUser, setAddUser] = useState('')

  const adminGroups = infraAdmins?.map(admin => admin.group)

  // Don't allow deleting the last group
  const hideRemoveGroupBtn =
    !admin || (infraAdmins?.length === 1 && adminGroups.includes(group?.id))

  return (
    <div className='mb-10'>
      <Head>
        <title>{group?.name} - Infra</title>
      </Head>

      {/* Header */}
      <header className='mt-6 mb-12 flex flex-col justify-between md:flex-row md:items-center'>
        <h1 className='truncate py-1 font-display text-xl font-medium'>
          <Link href='/groups'>
            <a className='text-gray-500/75 hover:text-gray-600'>Groups</a>
          </Link>{' '}
          <span className='mx-2 font-light text-gray-400'> / </span>{' '}
          {group?.name}
        </h1>

        {!hideRemoveGroupBtn && (
          <div className='my-3 flex space-x-2 md:my-0'>
            <RemoveButton
              onRemove={async () => {
                await fetch(`/api/groups/${id}`, {
                  method: 'DELETE',
                })

                router.replace('/groups')
              }}
              modalTitle='Remove group'
              modalMessage={
                <div className='truncate'>
                  Are you sure you want to remove{' '}
                  <span className='font-bold'>{group?.name}</span>?
                </div>
              }
            >
              Remove group
            </RemoveButton>
          </div>
        )}
      </header>

      {/* Users */}
      <h2 className='text-lg font-medium'>Users</h2>
      <div className='my-2.5 flex'>
        <div className='flex w-full flex-col rounded-lg border border-gray-200/75 px-4 pt-3 pb-4'>
          <h3 className='mb-2 text-sm font-medium'>Add user to group</h3>
          <div className='flex w-full space-x-2 '>
            <ComboBox
              options={allUsers
                ?.filter(au => !users?.find(u => au.name === u.name))
                ?.map(au => au.name)}
              selected={addUser}
              setSelected={setAddUser}
            />
            <button
              onClick={async () => {
                const user = allUsers?.find(au => au.name === addUser)

                if (!user) {
                  return false
                }

                await fetch(`/api/groups/${group?.id}/users`, {
                  method: 'PATCH',
                  body: JSON.stringify({ usersToAdd: [user.id] }),
                })

                // TODO: show optimistic results
                mutateUsers()
                mutate()
                setAddUser('')
              }}
              className='flex-none rounded-md bg-black px-4 py-[7px] text-xs font-medium text-white shadow-sm hover:bg-gray-700'
            >
              Add User
            </button>
          </div>
        </div>
      </div>
      <Table
        pageIndex={page - 1}
        pageSize={limit}
        pageCount={totalPages}
        count={totalCount}
        data={users}
        empty='No users'
        onPageChange={({ pageIndex }) => {
          router.push({
            pathname: router.pathname,
            query: { ...router.query, p: pageIndex + 1 },
          })
        }}
        columns={[
          {
            cell: info => (
              <span className='font-medium text-gray-700'>
                {info.getValue()}
              </span>
            ),
            header: () => <span>User</span>,
            accessorKey: 'name',
          },
          {
            cell: function Cell(info) {
              const [open, setOpen] = useState(false)

              return (
                <div className='text-right'>
                  <button
                    onClick={() => {
                      setOpen(true)
                    }}
                    className='p-1 text-2xs text-gray-500/75 hover:text-gray-600'
                  >
                    Remove
                    <span className='sr-only'>{info.row.original.name}</span>
                  </button>
                  <DeleteModal
                    open={open}
                    setOpen={setOpen}
                    onSubmit={async () => {
                      await fetch(`/api/groups/${group?.id}/users`, {
                        method: 'PATCH',
                        body: JSON.stringify({
                          usersToRemove: [info.row.original.id],
                        }),
                      })

                      // TODO: show optimistic result
                      mutateUsers()
                    }}
                    title='Remove user from this group?'
                    message='Are you sure you want to remove this user from the group?'
                  />
                </div>
              )
            },
            id: 'delete',
          },
        ]}
      />
    </div>
  )
}

GroupDetails.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
