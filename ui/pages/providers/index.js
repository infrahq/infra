import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import { useTable } from 'react-table'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'
import Remove from '../../components/remove'
import Metadata from '../../components/metadata'

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

function SidebarContent({ provider, admin, setSelectedProvider }) {
  const { mutate } = useSWRConfig()
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  const { name, url, clientID, created, updated } = provider

  const metadata = [
    { title: 'Name', data: name },
    { title: 'URL', data: url },
    { title: 'Client ID', data: clientID },
    {
      title: 'Added',
      data: created ? dayjs(created).fromNow() : '-',
    },
    {
      title: 'Updated',
      data: updated ? dayjs(updated).fromNow() : '-',
    },
  ]

  return (
    <div className='flex flex-1 flex-col space-y-6'>
      <section>
        <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
          Metadata
        </h3>
        <Metadata data={metadata} />
      </section>
      {admin && (
        <section className='flex flex-1 flex-col items-end justify-end py-6'>
          <Remove
            deleteModalOpen={deleteModalOpen}
            setDeleteModalOpen={setDeleteModalOpen}
            onSubmit={() => {
              mutate(
                '/api/providers',
                async ({ items: providers } = { items: [] }) => {
                  await fetch(`/api/providers/${provider.id}`, {
                    method: 'DELETE',
                  })

                  return { items: providers.filter(p => p?.id !== provider.id) }
                }
              )

              setDeleteModalOpen(false)
              setSelectedProvider(null)
            }}
            deleteModalTitle='Remove Identity Provider'
            deleteModalMessage={
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
  const { data: { items: providers } = {}, error } = useSWR('/api/providers')
  const { admin, loading: adminLoading } = useAdmin()
  const table = useTable({
    columns,
    data: providers?.sort((a, b) => b.created?.localeCompare(a.created)) || [],
  })

  const [selected, setSelected] = useState(null)

  const loading = adminLoading || (!providers && !error)

  return (
    <>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      {!loading && (
        <div className='flex h-full flex-1'>
          <div className='flex flex-1 flex-col space-y-4'>
            <PageHeader
              header='Providers'
              buttonHref='/providers/add'
              buttonLabel='Provider'
            />
            {error?.status ? (
              <div className='my-20 text-center text-sm font-light text-gray-300'>
                {error?.info?.message}
              </div>
            ) : (
              <div className='flex min-h-0 flex-1 flex-col overflow-y-scroll px-6'>
                <Table
                  {...table}
                  getRowProps={row => ({
                    onClick: () => setSelected(row.original),
                    className:
                      selected?.id === row.original.id
                        ? 'bg-gray-900/50'
                        : 'cursor-pointer',
                  })}
                />
                {providers?.length === 0 && (
                  <EmptyTable
                    title='There are no providers'
                    subtitle={
                      <>
                        Identity providers allow you to connect your existing
                        users &amp; groups to Infra.
                      </>
                    }
                    iconPath='/providers.svg'
                    buttonHref='/providers/add'
                    buttonText='Provider'
                  />
                )}
              </div>
            )}
          </div>
          {selected && (
            <Sidebar
              onClose={() => setSelected(null)}
              title={selected.name}
              iconPath='/providers.svg'
            >
              <SidebarContent
                provider={selected}
                admin={admin}
                setSelectedProvider={setSelected}
              />
            </Sidebar>
          )}
        </div>
      )}
    </>
  )
}

Providers.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
