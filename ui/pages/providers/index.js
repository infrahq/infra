import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import { useTable } from 'react-table'
import dayjs from 'dayjs'

import { kind } from '../../lib/providers'
import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/modals/delete'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'

const columns = [{
  Header: 'Name',
  accessor: p => p,
  Cell: ({ value: provider }) => (
    <div className='flex py-1.5 items-center'>
      <div className='border border-gray-800 flex-none flex items-center justify-center w-7 h-7 rounded-md'>
        {kind(provider.url)
          ? <img className='h-2' src={`/providers/${kind(provider.url)}.svg`} />
          : provider.name[0].toUpperCase()}
      </div>
      <div className='text-2xs leading-none ml-3'>{provider.name}</div>
    </div>
  )
}, {
  Header: 'URL',
  accessor: p => p,
  Cell: ({ value: provider }) => (
    <div className='text-3xs text-gray-400'>{provider.url}</div>
  )
}
]

function SidebarContent({ provider }) {
  return (
    <div className='flex-1 flex flex-col space-y-6'>
      <section>
        <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Metadata</h3>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Name</div>
            <div className='text-2xs'>{provider.name}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>URL</div>
            <div className='text-2xs'>{provider.url}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Client ID</div>
            <div className='text-2xs font-mono'>{provider.clientID}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Added</div>
            <div className='text-2xs'>{provider?.created ? dayjs(provider.created).fromNow() : '-'}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Updated</div>
            <div className='text-2xs'>{provider.updated ? dayjs(provider.updated).fromNow() : '-'}</div>
          </div>
        </div>
      </section>
    </div>
  )
}

export default function Providers () {
  const router = useRouter()

  const { data: { items: providers } = {}, error } = useSWR('/api/providers')
  const { admin, loading: adminLoading } = useAdmin()
  const { mutate } = useSWRConfig()
  const table = useTable({ columns, data: providers?.sort((a, b) => b.created?.localeCompare(a.created)) || [] })
  
  const [selectedProvider, setSelectedProvider] = useState(null)
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  const loading = adminLoading || (!providers && !error)

  function edit () {
    router.replace(`/providers/edit/details?id=${selectedProvider.id}`)
    return null
  }

  return (
    <>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      {!loading && (
        <div className='flex-1 flex h-full'>
          <div className='flex-1 flex flex-col space-y-4'>
            <PageHeader header='Providers' buttonHref='/providers/add' buttonLabel='Provider' />
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              : (
                <>
                  <Table 
                    {...table}
                    getRowProps={row => ({
                      onClick: () => setSelectedProvider(row.original),
                      style: {
                        cursor: 'pointer'
                      }
                    })}
                  />
                  {providers?.length === 0 &&
                    <EmptyTable
                      title='There are no providers'
                      subtitle={<>Identity providers allow you to connect your existing users &amp; groups to Infra.</>}
                      iconPath='/providers.svg'
                      buttonHref='/providers/add'
                      buttonText='Provider'
                    />}
                </>
                )}
          </div>
          {selectedProvider && 
            <Sidebar
              handleClose={() => setSelectedProvider(null)}
              title={selectedProvider.name}
              iconPath='/providers.svg'
              showActionBtn={admin}
              remove={() => setDeleteModalOpen(true)}
              edit={() => edit()}
            >
              <SidebarContent provider={selectedProvider} />
            </Sidebar>}
            <DeleteModal
              open={deleteModalOpen}
              setOpen={setDeleteModalOpen}
              onSubmit={() => {
                  mutate('/api/providers', async ({ items: providers } = { items: [] }) => {
                    await fetch(`/api/providers/${selectedProvider.id}`, {
                    method: 'DELETE'
                  })

                  return { items: providers.filter(p => p?.id !== selectedProvider.id) }
                })

                setDeleteModalOpen(false)
                setSelectedProvider(null)
              }}
              title='Remove Identity Provider'
              message={(<>Are you sure you want to delete <span className='font-bold text-white'>{selectedProvider?.name}</span>? This action cannot be undone.</>)}
            />
        </div>
      )}
    </>
  )
}

Providers.layout = function (page) {
  return (
    <Dashboard>
      {page}
    </Dashboard>
  )
}
