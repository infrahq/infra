import Head from 'next/head'
import { useRouter } from 'next/router'
import useSWR from 'swr'
import dayjs from 'dayjs'
import { Menu, Transition, Dialog } from '@headlessui/react'
import { Fragment, useState } from 'react'
import { DotsHorizontalIcon, XIcon } from '@heroicons/react/outline'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'

import { useServerConfig } from '../../lib/serverconfig'

import DeleteModal from '../../components/delete-modal'
import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'

function UsersAddDialog({ setOpen, onAdded = () => {} }) {
  const [email, setEmail] = useState('')
  const [success, setSuccess] = useState(false)
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const { isEmailConfigured } = useServerConfig()

  async function onSubmit(e) {
    e.preventDefault()

    setError('')

    try {
      const res = await fetch('/api/users', {
        method: 'POST',
        body: JSON.stringify({
          name: email,
        }),
      })
      const user = await res.json()

      if (!res.ok) {
        throw user
      }

      setSuccess(true)
      setPassword(user.oneTimePassword)
      onAdded(user)
    } catch (e) {
      if (e.code === 409) {
        setError('user with this email already exists')
        return false
      }

      setError(e.message)

      return false
    }
  }

  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='font-medium'>Add user</h1>
      <div className='space-y-4'>
        {success ? (
          <div className='flex flex-col'>
            {isEmailConfigured ? (
              <h2 className='mt-5 text-sm'>
                User added. The user has been emailed a link inviting them to
                join.
              </h2>
            ) : (
              <div>
                <h2 className='mt-5 text-sm'>
                  User added. Send the user this temporary password for their
                  initial login. This password will not be shown again.
                </h2>
                <div className='mt-6 flex flex-col space-y-3'>
                  <label className='text-2xs font-medium text-gray-700'>
                    Temporary Password
                  </label>
                  <span
                    readOnly
                    className='round-md my-0 w-full py-1 font-mono text-xs text-gray-600 focus:outline-none'
                  >
                    {password}
                  </span>
                </div>
              </div>
            )}
            <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
              <button
                onClick={() => {
                  setSuccess(false)
                  setEmail('')
                  setPassword('')
                }}
                className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
              >
                Add Another
              </button>
              <button
                onClick={() => {
                  setOpen(false)
                }}
                autoFocus
                className='inline-flex items-center rounded-md border border-transparent bg-black px-6 py-2 text-xs font-medium text-white hover:bg-gray-700'
              >
                Done
              </button>
            </div>
          </div>
        ) : (
          <form onSubmit={onSubmit} className='flex flex-col space-y-4'>
            <div className='mb-4 flex flex-col'>
              <div className='relative mt-4'>
                <label className='text-2xs font-medium text-gray-700'>
                  User Email
                </label>
                <input
                  autoFocus
                  spellCheck='false'
                  type='email'
                  value={email}
                  onChange={e => {
                    setEmail(e.target.value)
                    setError('')
                  }}
                  className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                    error ? 'border-red-500' : 'border-gray-300'
                  }`}
                />
                {error && (
                  <p className='absolute text-xs text-red-500'>{error}</p>
                )}
              </div>
            </div>
            <div className='flex flex-row items-center justify-end space-x-3'>
              <button
                type='button'
                onClick={() => setOpen(false)}
                className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
              >
                Cancel
              </button>
              <button
                type='submit'
                className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2.5 text-xs font-medium text-white shadow-sm hover:bg-gray-800'
              >
                Add User
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}

export default function Users() {
  const router = useRouter()
  const page = Math.max(parseInt(router.query.p) || 1, 1)
  const limit = 20
  const { data: { items: users, totalPages, totalCount } = {}, mutate } =
    useSWR(`/api/users?page=${page}&limit=${limit}`)
  const [open, setOpen] = useState(false)

  const { data: { items: providers } = {} } = useSWR(`/api/providers?limit=999`)
  const { data: auth } = useSWR('/api/users/self')

  return (
    <div className='mb-10'>
      <Head>
        <title>Users - Infra</title>
      </Head>

      {/* Header */}
      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 text-xl font-medium'>Users</h1>
        <button
          onClick={() => setOpen(true)}
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'
        >
          Add user
        </button>

        {/* Add dialog */}
        <Transition.Root show={open} as={Fragment}>
          <Dialog as='div' className='relative z-50' onClose={setOpen}>
            <Transition.Child
              as={Fragment}
              enter='ease-out duration-150'
              enterFrom='opacity-0'
              enterTo='opacity-100'
              leave='ease-in duration-100'
              leaveFrom='opacity-100'
              leaveTo='opacity-0'
            >
              <div className='fixed inset-0 bg-white bg-opacity-75 backdrop-blur-xl transition-opacity' />
            </Transition.Child>
            <div className='fixed inset-0 z-10 overflow-y-auto'>
              <div className='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
                <Transition.Child
                  as={Fragment}
                  enter='ease-out duration-150'
                  enterFrom='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                  enterTo='opacity-100 translate-y-0 sm:scale-100'
                  leave='ease-in duration-100'
                  leaveFrom='opacity-100 translate-y-0 sm:scale-100'
                  leaveTo='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                >
                  <Dialog.Panel className='relative w-full transform overflow-hidden rounded-xl border border-gray-100 bg-white p-8 text-left shadow-xl shadow-gray-300/10 transition-all sm:my-8 sm:max-w-sm'>
                    <UsersAddDialog
                      setOpen={setOpen}
                      onAdded={() => {
                        mutate()
                      }}
                    />
                  </Dialog.Panel>
                </Transition.Child>
              </div>
            </div>
          </Dialog>
        </Transition.Root>
      </header>

      {/* Table */}
      <Table
        onPageChange={({ pageIndex }) => {
          router.push({
            pathname: router.pathname,
            query: { ...router.query, p: pageIndex + 1 },
          })
        }}
        count={totalCount}
        pageCount={totalPages}
        pageIndex={parseInt(page) - 1}
        pageSize={limit}
        data={users}
        columns={[
          {
            cell: info => (
              <div className='truncate py-1 font-medium text-gray-700'>
                {info.getValue()}
              </div>
            ),
            header: <span>Name</span>,
            accessorKey: 'name',
          },
          {
            cell: info => (
              <div className='truncate'>
                {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
              </div>
            ),
            header: () => <span>Last&nbsp;seen</span>,
            accessorKey: 'lastSeenAt',
          },
          {
            cell: info => (
              <div className='truncate'>
                {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
              </div>
            ),
            header: () => (
              <span className='hidden truncate md:table-cell'>Added</span>
            ),
            accessorKey: 'created',
          },
          {
            cell: info => (
              <div className='flex space-x-1'>
                {info.getValue().map(pn => {
                  if (pn === 'infra') {
                    return (
                      <img
                        alt='provider icon'
                        title={pn}
                        key={pn}
                        className='translate-[-50%] h-3.5'
                        src={`/icon.svg`}
                      />
                    )
                  } else {
                    const provider = providers?.find(p => p.name === pn)
                    if (!provider) {
                      return null
                    }

                    return (
                      <img
                        alt='provider icon'
                        title={pn}
                        key={pn}
                        className='translate-[-50%] h-3.5'
                        src={`/providers/${provider.kind}.svg`}
                      />
                    )
                  }
                })}
              </div>
            ),
            header: () => <span>Providers</span>,
            accessorKey: 'providerNames',
          },
          {
            id: 'actions',
            cell: function Cell(info) {
              const [open, setOpen] = useState(false)
              let [referenceElement, setReferenceElement] = useState()
              let [popperElement, setPopperElement] = useState()
              let { styles, attributes } = usePopper(
                referenceElement,
                popperElement,
                {
                  placement: 'bottom-end',
                  modifiers: [
                    {
                      name: 'flip',
                      enabled: false,
                    },
                  ],
                }
              )

              if (info.row.original.id === auth?.id) {
                return null
              }

              return (
                <div className='flex justify-end'>
                  <Menu as='div' className='relative inline-block text-left'>
                    <Menu.Button
                      ref={setReferenceElement}
                      className='cursor-pointer rounded-md border border-transparent px-1 text-gray-400 hover:bg-gray-50 hover:text-gray-600 group-hover:border-gray-200 group-hover:text-gray-500 group-hover:shadow-md group-hover:shadow-gray-300/20'
                    >
                      <DotsHorizontalIcon className='z-0 h-[18px]' />
                    </Menu.Button>
                    {ReactDOM.createPortal(
                      <div
                        ref={setPopperElement}
                        style={styles.popper}
                        {...attributes.popper}
                      >
                        <Transition
                          as={Fragment}
                          enter='transition ease-out duration-100'
                          enterFrom='transform opacity-0 scale-95'
                          enterTo='transform opacity-100 scale-100'
                          leave='transition ease-in duration-75'
                          leaveFrom='transform opacity-100 scale-100'
                          leaveTo='transform opacity-0 scale-95'
                        >
                          <Menu.Items className='absolute right-0 z-10 mt-2 w-40 origin-top-right divide-y divide-gray-100 rounded-md bg-white shadow-lg shadow-gray-300/20 ring-1 ring-black ring-opacity-5 focus:outline-none'>
                            <div className='px-1 py-1'>
                              <Menu.Item>
                                {({ active }) => (
                                  <button
                                    className={`${
                                      active ? 'bg-gray-50' : 'bg-white'
                                    } group flex w-full items-center rounded-md px-2 py-1.5 text-xs font-medium text-red-500`}
                                    onClick={() => setOpen(true)}
                                  >
                                    <XIcon className='mr-1 mt-px h-3.5 w-3.5' />{' '}
                                    Remove user
                                  </button>
                                )}
                              </Menu.Item>
                            </div>
                          </Menu.Items>
                        </Transition>
                      </div>,
                      document.querySelector('body')
                    )}
                  </Menu>
                  <DeleteModal
                    open={open}
                    setOpen={setOpen}
                    primaryButtonText='Remove'
                    onSubmit={async () => {
                      await fetch(`/api/users/${info.row.original.id}`, {
                        method: 'DELETE',
                      })
                      setOpen(false)

                      mutate()
                    }}
                    title='Remove User'
                    message={
                      <>
                        Are you sure you want to remove{' '}
                        <span className='font-bold'>
                          {info.row.original.name}
                        </span>
                        ?
                      </>
                    }
                  />
                </div>
              )
            },
          },
        ]}
      />
    </div>
  )
}

Users.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
