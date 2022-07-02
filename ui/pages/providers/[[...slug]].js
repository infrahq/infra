import useSWR, { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'
import { useState } from 'react'
import Head from 'next/head'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'

const columns = [
  {
    Header: 'Name',
    accessor: p => p,
    Cell: ({ value: provider }) => (
      <div className='flex items-center py-1.5'>
        <div className='flex h-7 w-7 flex-none items-center justify-center rounded-md border border-gray-800'>
          <img
            alt='provider icon'
            className='h-2'
            src={`/providers/${provider.kind}.svg`}
          />
        </div>
        <div className='ml-3 text-2xs leading-none'>{provider.name}</div>
      </div>
    ),
  },
  {
    Header: 'URL',
    accessor: p => p,
    Cell: ({ value: provider }) => (
      <div className='text-3xs text-gray-400'>{provider.url}</div>
    ),
  },
]

function SidebarContent({ provider, admin }) {
  const { mutate } = useSWRConfig()
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const router = useRouter()

  return (
    <div className='flex flex-1 flex-col space-y-6'>
      <section>
        <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
          Metadata
        </h3>
        <div className='flex flex-col space-y-2 pt-3'>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Name</div>
            <div className='text-2xs'>{provider.name}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>URL</div>
            <div className='text-2xs'>{provider.url}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Client ID</div>
            <div className='text-2xs'>{provider.clientID}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Added</div>
            <div className='text-2xs'>
              {provider?.created ? dayjs(provider.created).fromNow() : '-'}
            </div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Updated</div>
            <div className='text-2xs'>
              {provider.updated ? dayjs(provider.updated).fromNow() : '-'}
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
            onSubmit={() => {
              router.replace('/providers')

              mutate(
                '/api/providers',
                async ({ items: providers } = { items: [] }) => {
                  fetch(`/api/providers/${provider.id}`, {
                    method: 'DELETE',
                  })

                  return { items: providers.filter(p => p?.id !== provider.id) }
                }
              )

              setDeleteModalOpen(false)
            }}
            title='Remove Identity Provider'
            message={
              <>
                Are you sure you want to delete{' '}
                <span className='font-bold text-white'>{provider?.name}</span>?
                This action cannot be undone.
              </>
            }
          />
        </section>
      )}
    </div>
  )
}

export default function Providers() {
  const { data: { items: providers } = {} } = useSWR('/api/providers')
  const { admin, loading: adminLoading } = useAdmin()

  const router = useRouter()

  if (adminLoading || !providers) {
    return null
  }

  const { slug: [id] = [] } = router.query
  const provider = providers?.find(p => p.id === id)
  if (id && !provider) {
    router.replace('/providers')
    return null
  }

  return (
    <>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      <div className='flex h-full flex-1'>
        <div className='flex flex-1 flex-col space-y-4'>
          <PageHeader
            header='Providers'
            buttonHref='/providers/add'
            buttonLabel='Provider'
          />
          <div className='flex min-h-0 flex-1 flex-col overflow-y-scroll px-6'>
            <Table
              columns={columns}
              data={
                providers?.sort((a, b) =>
                  b.created?.localeCompare(a.created)
                ) || []
              }
              getRowProps={row => ({
                onClick: () => router.push(`/providers/${row.original.id}`),
                className:
                  id === row.original.id ? 'bg-gray-900/50' : 'cursor-pointer',
              })}
            />
            {providers?.length === 0 && (
              <EmptyTable
                title='There are no providers'
                subtitle={
                  <>
                    Identity providers allow you to connect your existing users
                    &amp; groups to Infra.
                  </>
                }
                iconPath='/providers.svg'
                buttonHref='/providers/add'
                buttonText='Provider'
              />
            )}
          </div>
        </div>
        {id && (
          <Sidebar
            handleClose={() => router.push('/providers')}
            title={provider?.name}
            iconPath='/providers.svg'
          >
            <SidebarContent provider={provider} admin={admin} />
          </Sidebar>
        )}
      </div>
    </>
  )
}

Providers.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
