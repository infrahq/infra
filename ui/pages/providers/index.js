import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { Fragment, useState } from 'react'
import dayjs from 'dayjs'
import { usePopper } from 'react-popper'
import { Menu, Transition } from '@headlessui/react'
import * as ReactDOM from 'react-dom'

import { DotsHorizontalIcon, XIcon, PencilIcon } from '@heroicons/react/outline'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'

export default function Providers() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 999
  const { data: { items: providers } = {}, mutate } = useSWR(
    `/api/providers?page=${page}&limit=${limit}`
  )

  return (
    <div className='mb-10'>
      <Head>
        <title>Providers - Infra</title>
      </Head>

      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 font-display text-xl font-medium'>Providers</h1>
        <Link href='/providers/add' data-testid='page-header-button-link'>
          <button className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'>
            Connect provider
          </button>
        </Link>
      </header>

      <Table
        data={providers}
        empty='No providers'
        columns={[
          {
            cell: info => (
              <div className='flex flex-row items-center py-1'>
                <div className='mr-3 flex h-9 w-9 flex-none items-center justify-center rounded-md border border-gray-200'>
                  <img
                    alt='provider icon'
                    className='h-4'
                    src={`/providers/${info.row.original.kind}.svg`}
                  />
                </div>
                <div className='flex flex-col'>
                  <div className='text-sm font-medium text-gray-700'>
                    {info.getValue()}
                  </div>
                  <div className='text-2xs text-gray-500 sm:hidden'>
                    {info.row.original.url}
                  </div>
                  <div className='font-mono text-2xs text-gray-400 lg:hidden'>
                    {info.row.original.clientID}
                  </div>
                </div>
              </div>
            ),
            header: () => <span>Name</span>,
            accessorKey: 'name',
          },
          {
            cell: info => (
              <div className='truncate'>
                {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
              </div>
            ),
            header: () => (
              <span className='hidden truncate lg:table-cell'>Added</span>
            ),
            accessorKey: 'created',
          },
          {
            cell: info => (
              <div className='hidden sm:table-cell'>{info.getValue()}</div>
            ),
            header: () => <span className='hidden sm:table-cell'>URL</span>,
            accessorKey: 'url',
          },
          {
            cell: info => (
              <div className='hidden font-mono lg:table-cell'>
                {info.getValue()}
              </div>
            ),
            header: () => (
              <span className='hidden lg:table-cell'>Client ID</span>
            ),
            accessorKey: 'clientID',
          },
          {
            id: 'actions',
            cell: function Cell(info) {
              const [deleteOpen, setDeleteOpen] = useState(false)
              const [referenceElement, setReferenceElement] = useState(null)
              const [popperElement, setPopperElement] = useState(null)
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
                                    } group flex w-full items-center rounded-md px-2 py-1.5 text-xs font-medium text-gray-600`}
                                    onClick={() =>
                                      router.replace(
                                        `/providers/${info.row.original.id}`
                                      )
                                    }
                                  >
                                    <PencilIcon className='mr-1 mt-px h-3.5 w-3.5' />{' '}
                                    Edit
                                  </button>
                                )}
                              </Menu.Item>
                              <Menu.Item>
                                {({ active }) => (
                                  <button
                                    className={`${
                                      active ? 'bg-gray-50' : 'bg-white'
                                    } group flex w-full items-center rounded-md px-2 py-1.5 text-xs font-medium text-red-500`}
                                    onClick={() => setDeleteOpen(true)}
                                  >
                                    <XIcon className='mr-1 mt-px h-3.5 w-3.5' />{' '}
                                    Remove
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
                    open={deleteOpen}
                    setOpen={setDeleteOpen}
                    onSubmit={async () => {
                      await fetch(`/api/providers/${info.row.original.id}`, {
                        method: 'DELETE',
                      })
                      setDeleteOpen(false)

                      mutate({
                        items: providers.filter(
                          p => p.id !== info.row.original.id
                        ),
                      })
                    }}
                    title='Remove Identity Provider'
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

Providers.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
