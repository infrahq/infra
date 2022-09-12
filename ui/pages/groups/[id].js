import { useRouter } from 'next/router'
import Link from 'next/link'
import Head from 'next/head'
import useSWR from 'swr'
import { useState, useEffect } from 'react'
import { ChevronDownIcon, CheckIcon } from '@heroicons/react/outline'
import { Combobox as HeadlessUIComboBox } from '@headlessui/react'

import { useAdmin } from '../../lib/admin'

import Notification from '../../components/notification'
import Table from '../../components/table'
import RemoveButton from '../../components/remove-button'
import DeleteModal from '../../components/delete-modal'
import Dashboard from '../../components/layouts/dashboard'

function ComboBox({ options = [], selected, setSelected }) {
  const [query, setQuery] = useState('')

  const filtered = options?.filter(u =>
    u.toLowerCase().includes(query.toLowerCase())
  )

  return (
    <HeadlessUIComboBox
      as='div'
      value={selected}
      onChange={user => {
        setSelected(user)
      }}
    >
      <div className='relative'>
        <HeadlessUIComboBox.Input
          className='w-full rounded-md border border-gray-300 bg-white py-[7px] pl-2.5 pr-10 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 md:text-xs'
          value={query}
          placeholder='Add user to group'
          onChange={event => setQuery(event.target.value)}
        />
        <HeadlessUIComboBox.Button className='absolute inset-y-0 right-0 flex items-center rounded-r-md px-2 focus:outline-none'>
          <ChevronDownIcon
            className='h-4 w-4 text-gray-400'
            aria-hidden='true'
          />
        </HeadlessUIComboBox.Button>

        {filtered?.length > 0 && (
          <HeadlessUIComboBox.Options className='absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none md:text-xs'>
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
                          className='h-4 w-4 text-gray-400'
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
  const created = router.query.created
  const page = Math.max(parseInt(router.query.p) || 1, 1)
  const limit = 999

  const { data: group, mutate: mutate } = useSWR(`/api/groups/${id}`)
  const { admin } = useAdmin()

  const {
    data: { items: users, totalCount, totalPages } = {},
    mutate: mutateUsers,
  } = useSWR(`/api/users?group=${group?.id}&limit=${limit}&p=${page}`)

  const { data: { items: allUsers } = {} } = useSWR(`/api/users?limit=999`)

  const { data: { items: infraAdmins } = {} } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=999'
  )

  const [showCreated, setShowCreated] = useState(false)

  useEffect(() => {
    if (created) {
      setShowCreated(true)
      setTimeout(() => setShowCreated(false), 3000)
    }
  }, [created])

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

      {/* Created notification */}
      {created && (
        <Notification
          show={showCreated}
          setShow={setShowCreated}
          text='Group added'
        />
      )}

      {/* Header */}
      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 text-xl font-medium'>
          <Link href='/groups'>
            <a className='text-gray-500/75 hover:text-gray-600'>Groups</a>
          </Link>{' '}
          <span className='mx-2 font-light text-gray-400'> / </span>{' '}
          {group?.name}
        </h1>

        {!hideRemoveGroupBtn && (
          <RemoveButton
            onRemove={async () => {
              await fetch(`/api/groups/${id}`, {
                method: 'DELETE',
              })

              router.replace('/groups')
            }}
            modalTitle='Remove group'
            modalMessage={
              <>
                Are you sure you want to delete{' '}
                <span className='font-bold'>{group?.name}</span>? This action
                cannot be undone.
              </>
            }
          >
            Remove group
          </RemoveButton>
        )}
      </header>

      {/* Users */}
      <div className='my-2.5 flex justify-between'>
        <h2 className='py-2 text-lg font-medium text-gray-600'>Users</h2>
        <div className='flex items-center space-x-2'>
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
            className='rounded-md bg-black px-4 py-[7px] text-xs font-medium text-white shadow-sm hover:bg-gray-700'
          >
            Add
          </button>
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
