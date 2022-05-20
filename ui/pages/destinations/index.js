import useSWR, { useSWRConfig } from 'swr'
import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import Loader from '../../components/loader'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import DeleteModal from '../../components/modals/delete'
import Grant from '../../components/grant'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'

const columns = [{
  Header: 'Name',
  accessor: 'name',
  Cell: ({ value }) => (
    <div className='flex py-1.5 items-center'>
      <div className='border border-gray-800 flex-none flex items-center justify-center w-7 h-7 mr-3 rounded-md'>
        <img className='opacity-25' src='/row-infrastructure.svg' />
      </div>
      {value}
    </div>
  )
}, {
  Header: 'Kind',
  id: 'kind',
  Cell: () => <span className='text-gray-400'>Cluster</span>
}, {
  id: 'connected',
  Header: () => (
    <div className='text-right'>Connection</div>
  ),
  accessor: 'updated',
  Cell: ({ value: updated }) => {
    const connected = (new Date() - new Date(updated)) < 24 * 60 * 60 * 1000
    return (
      <div className='flex items-center text-gray-400 justify-end'>
        {connected
          ? (
            <>
              <div className='w-[7px] h-[7px] bg-green-400 rounded-full mr-1.5' />
              Connected
            </>
            )
          : (
            <div className='flex items-center'>
              <div className='w-[7px] h-[7px] bg-gray-600 rounded-full mr-1.5' />
              Disconnected
            </div>
            )}
      </div>
    )
  }
}]

function SlideContent ({ destination, isAdmin, setSelectedDestination }) {
  const { mutate } = useSWRConfig()
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  return (
    <>
      {isAdmin &&
        <section>
          <div className='border-b border-gray-800 mt-4'>
            <div className='text-xxs text-gray-400 uppercase pb-5'>Access</div>
          </div>
          <div className='pt-3 pb-12'>
            <Grant id={destination.id} />
          </div>
        </section>}
      <section>
        <div className='border-b border-gray-800 mt-4'>
          <div className='text-xxs text-gray-400 uppercase pb-5'>Meta</div>
        </div>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-xs w-1/3'>ID</div>
            <div className='text-xs font-mono'>{destination.id}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-xs w-1/3'>Added</div>
            <div className='text-xs'>{dayjs(destination?.created).fromNow()}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-xs w-1/3'>Last Seen</div>
            <div className='text-xs'>{dayjs(destination?.lastSeen).fromNow()}</div>
          </div>
        </div>
      </section>
      <section className='flex-1 flex flex-col items-end flex-shrink-0 justify-end py-6'>
        <button
          type='button'
          onClick={() => setDeleteModalOpen(true)}
          className='border border-violet-300 rounded-md flex items-center text-xs px-6 py-3 text-violet-100'
        >
          Delete
        </button>
        <DeleteModal
          open={deleteModalOpen}
          onCancel={() => setDeleteModalOpen(false)}
          onSubmit={async () => {
            mutate('/v1/destinations', async destinations => {
              await fetch(`/v1/destinations/${destination.id}`, {
                method: 'DELETE'
              })

              return destinations?.filter(d => d?.id !== destination.id)
            })

            setDeleteModalOpen(false)
            setSelectedDestination(null)
          }}
          title='Delete Cluster'
          message={<>Are you sure you want to disconnect <span className='text-white font-bold'>{destination?.name}?</span><br />Note: you must also uninstall the Infra Connector from this cluster.</>}
        />
      </section>
    </>
  )
}

export default function Destinations () {
  const { data: destinations, error } = useSWR('/v1/destinations')
  const { admin, loading: adminLoading } = useAdmin()
  const [selectedDestination, setSelectedDestination] = useState(null)
  const table = useTable({ columns, data: destinations || [] })

  const loading = adminLoading || (!destinations && !error)

  return (
    <>
      <Head>
        <title>Destinations - Infra</title>
      </Head>
      {loading
        ? (<Loader />)
        : (
          <div className='flex-1 flex h-full'>
            <main className='flex-1 space-y-8'>
              <PageHeader header='Infrastructure' buttonHref={admin && '/destinations/add'} buttonLabel='Infrastructure' />
              {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              : (
                <div>
                  {<Table
                    {...table}
                    getRowProps={row => ({
                      onClick: () => setSelectedDestination(row.original),
                      style: {
                        cursor: 'pointer'
                      }
                    })}
                  />}
                  {destinations?.length === 0 && <EmptyTable
                    title='There is no infrastructure'
                    subtitle='There is currently no infrastructure connected to Infra'
                    iconPath='/destinations.svg'
                    buttonHref={admin && '/destinations/add'}
                    buttonText='Infrastructure'
                  />}
                </div>
                )}
            </main>
            {selectedDestination &&
              <Sidebar
                handleClose={() => setSelectedDestination(null)}
                title={selectedDestination.name}
                iconPath='/destinations.svg'
              >
                <SlideContent destination={selectedDestination} isAdmin={admin} setSelectedDestination={setSelectedDestination} />
              </Sidebar>}
          </div>
          )}
    </>
  )
}

Destinations.layout = function (page) {
  return (
    <Dashboard>
      {page}
    </Dashboard>
  )
}
