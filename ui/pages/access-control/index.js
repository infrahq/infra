import Head from 'next/head'
import { useEffect, useState, Fragment } from 'react'

import useSWR from 'swr'
import dayjs from 'dayjs'
import { TrashIcon } from '@heroicons/react/24/outline'
import { Dialog, Transition } from '@headlessui/react'

import Dashboard from '../../components/layouts/dashboard'
import Table from '../../components/table'
import { useUser } from '../../lib/hooks'

function CreateAccessDialog({ setOpen }) {
  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='py-1 font-display text-lg font-medium'>Create access</h1>
      <div className='space-y-4'></div>
    </div>
  )
}

export default function AccessControl() {
  const { data: { items: users } = {} } = useSWR('/api/users?limit=1000')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=1000')
  const { data: { items: allGrants } = {}, mutate } = useSWR(
    `/api/grants?limit=1000`
  )

  const { isAdmin } = useUser()

  const [grants, setGrants] = useState({})
  const [openCreateAccess, setOpenCreateAccess] = useState(false)

  useEffect(() => {
    setGrants(
      allGrants
        ?.filter(g => g.resource !== 'infra')
        .map(g => {
          if (g.group) {
            return { ...g, type: 'group', identityId: g.group }
          }

          if (g.user) {
            return { ...g, type: 'user', identityId: g.user }
          }

          return g
        })
    )
  }, [allGrants])

  return (
    <div className='mb-10'>
      <Head>
        <title>Access Control - Infra</title>
      </Head>
      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 font-display text-xl font-medium'>
          Access Control
        </h1>
        {isAdmin && (
          <>
            <button
              onClick={() => setOpenCreateAccess(true)}
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
            >
              Create access
            </button>
            <Transition.Root show={openCreateAccess} as={Fragment}>
              <Dialog
                as='div'
                className='relative z-50'
                onClose={setOpenCreateAccess}
              >
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
                <div className='fixed inset-0 z-30 overflow-y-auto'>
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
                        <CreateAccessDialog setOpen={setOpenCreateAccess} />
                      </Dialog.Panel>
                    </Transition.Child>
                  </div>
                </div>
              </Dialog>
            </Transition.Root>
          </>
        )}
      </header>
      <Table
        data={grants}
        allowDelete={isAdmin}
        onDelete={async selectedIds => {
          const promises = selectedIds.map(
            async selectedId =>
              await fetch(`/api/grants/${selectedId}`, { method: 'DELETE' })
          )

          await Promise.all(promises)
          mutate()
        }}
        columns={[
          {
            header: <span>User / Group </span>,
            id: 'identity',
            accessorKey: 'identityId',
            cell: function Cell(info) {
              const name =
                users?.find(u => u.id === info.row.original.identityId)?.name ||
                groups?.find(g => g.id === info.row.original.identityId)?.name

              return (
                <div className='flex flex-col'>
                  <div className='text-sm font-medium text-gray-700'>
                    {name}
                  </div>
                  <div className='text-2xs text-gray-500'>
                    {info.row.original.type}
                  </div>
                </div>
              )
            },
          },
          {
            cell: info => <span>{info.getValue()}</span>,
            header: <span>Role</span>,
            accessorKey: 'privilege',
          },
          {
            cell: info => <span>{info.getValue()}</span>,
            header: () => <span>Resource</span>,
            accessorKey: 'resource',
          },
          {
            cell: info => (
              <div className='hidden sm:table-cell'>
                {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
              </div>
            ),
            header: () => <span>Created</span>,
            accessorKey: 'created',
          },
          {
            id: 'delete',
            cell: function Cell(info) {
              return (
                isAdmin && (
                  <button
                    type='button'
                    onClick={async () => {
                      await fetch(`/api/grants/${info.row.original.id}`, {
                        method: 'DELETE',
                      })
                      mutate()
                    }}
                    className='group flex w-full items-center rounded-md bg-white px-2 py-1.5 text-right text-xs font-medium text-red-500 hover:text-red-500/50'
                  >
                    <TrashIcon className='mr-2 h-3.5 w-3.5' />
                    <span className='hidden sm:block'>Remove</span>
                  </button>
                )
              )
            },
          },
        ]}
      />
    </div>
  )
}

AccessControl.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
