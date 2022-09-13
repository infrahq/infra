import Head from 'next/head'
import useSWR from 'swr'
import { useRouter } from 'next/router'
import dayjs from 'dayjs'
import { Transition, Dialog } from '@headlessui/react'
import { Fragment, useState } from 'react'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'

function AddGroupsDialog({ setOpen, onAdded = () => {} }) {
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function onSubmit(e) {
    e.preventDefault()

    setError('')

    try {
      const res = await fetch('/api/groups', {
        method: 'POST',
        body: JSON.stringify({ name }),
      })

      const group = await res.json()

      if (!res.ok) {
        throw group
      }

      onAdded(group)
      setOpen(false)
    } catch (e) {
      setError(e.message)
    }

    setSubmitting(false)

    return false
  }

  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='py-1 text-lg font-medium'>Add group</h1>
      <form className='my-2 flex flex-col' onSubmit={onSubmit}>
        <label className='text-2xs font-medium text-gray-700'>Name</label>
        <div className='relative mb-4'>
          <input
            name='name'
            required
            autoFocus
            spellCheck='false'
            type='search'
            value={name}
            onChange={e => setName(e.target.value)}
            className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
              error ? 'border-red-500' : 'border-gray-300'
            }`}
          />
          {error && (
            <p className='absolute mt-1 text-xs text-red-500'>{error}</p>
          )}
        </div>
        <div className='space-x-2 self-end'>
          <button
            type='button'
            onClick={() => setOpen(false)}
            className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
          >
            Cancel
          </button>
          <button
            disabled={submitting}
            className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'
          >
            Add group
          </button>
        </div>
      </form>
    </div>
  )
}

export default function Groups() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 10
  const { data: { items: groups, totalPages, totalCount } = {}, mutate } =
    useSWR(`/api/groups?page=${page}&limit=${limit}`)
  const [open, setOpen] = useState(false)

  return (
    <div className='mb-10'>
      <Head>
        <title>Groups - Infra</title>
      </Head>

      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 text-xl font-medium'>Groups</h1>
        {/* Add dialog */}
        <button
          onClick={() => setOpen(true)}
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'
        >
          Add group
        </button>
        <Transition.Root show={open} as={Fragment}>
          <Dialog as='div' className='relative z-10' onClose={setOpen}>
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
                  <Dialog.Panel className='relative w-full transform overflow-hidden rounded-xl border border-gray-100 bg-white px-8 py-4 text-left shadow-xl shadow-gray-300/10 transition-all sm:max-w-sm'>
                    <AddGroupsDialog
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
      <div className='flex min-h-0 flex-1 flex-col'>
        <Table
          href={row => `/groups/${row.original.id}`}
          count={totalCount}
          pageCount={totalPages}
          pageIndex={parseInt(page) - 1}
          pageSize={limit}
          data={groups}
          empty='No groups'
          onPageChange={({ pageIndex }) => {
            router.push({
              pathname: router.pathname,
              query: { ...router.query, p: pageIndex + 1 },
            })
          }}
          columns={[
            {
              cell: info => (
                <div className='flex flex-col'>
                  <div className='text-sm font-medium text-gray-700'>
                    {info.getValue()}
                  </div>
                  <div className='text-2xs text-gray-500 sm:hidden'>
                    {info.row.original.totalUsers || 'No'}{' '}
                    {info.row.original.totalUsers === 1 ? 'user' : 'users'}
                  </div>
                </div>
              ),
              header: () => <span>Name</span>,
              accessorKey: 'name',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() || 'No'}{' '}
                  {info.getValue() === 1 ? 'user' : 'users'}
                </div>
              ),
              header: () => <span className='hidden sm:table-cell'>Users</span>,
              accessorKey: 'totalUsers',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
                </div>
              ),
              header: () => <span className='hidden sm:table-cell'>Added</span>,
              accessorKey: 'created',
            },
          ]}
        />
      </div>
    </div>
  )
}

Groups.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
