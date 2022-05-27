import useSWR, { useSWRConfig } from 'swr'
import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import DeleteModal from '../../components/modals/delete'
import Grant from '../../components/grant'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'

function SidebarContent ({ destination, admin, setSelectedDestination }) {
  const { data: auth } = useSWR('/api/users/self')
  const { data: { items: grants } = {} } = useSWR(() => `/api/grants?user=${auth.id}&resource=${destination.name}`)

  const { mutate } = useSWRConfig()
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  return (
    <div className='flex-1 flex flex-col space-y-6'>
      {grants?.length > 0 && (
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Connect</h3>
          <p className='text-2xs my-4'>Connect to this cluster via the <a target='_blank' href='https://infrahq.com/docs/install/install-infra-cli' className='underline text-violet-200 font-medium' rel='noreferrer'>Infra CLI</a></p>
          <pre className='px-4 py-3 rounded-md text-gray-300 bg-gray-900 text-2xs leading-normal'>
            infra login {window.location.host}<br />
            infra use {destination.name}<br />
            kubectl get pods
          </pre>
        </section>
      )}
      {admin &&
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Access</h3>
          <Grant id={destination.id} />
        </section>}
      <section>
        <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Meta</h3>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>ID</div>
            <div className='text-2xs font-mono'>{destination.id}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Added</div>
            <div className='text-2xs'>{destination?.created ? dayjs(destination.created).fromNow() : '-'}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Updated</div>
            <div className='text-2xs'>{destination.updated ? dayjs(destination.updated).fromNow() : '-'}</div>
          </div>
        </div>
      </section>
      <section className='flex-1 flex flex-col items-end justify-end py-6'>
        <button
          type='button'
          onClick={() => setDeleteModalOpen(true)}
          className='border border-violet-300 rounded-md flex items-center text-2xs px-6 py-3 text-violet-100'
        >
          Remove
        </button>
        <DeleteModal
          open={deleteModalOpen}
          setOpen={setDeleteModalOpen}
          onSubmit={async () => {
            mutate('/api/destinations', async ({ items: destinations } = { items: [] }) => {
              await fetch(`/api/destinations/${destination.id}`, {
                method: 'DELETE'
              })

              return { items: destinations.filter(d => d?.id !== destination.id) }
            })

            setDeleteModalOpen(false)
            setSelectedDestination(null)
          }}
          title='Remove Cluster'
          message={<>Are you sure you want to disconnect <span className='text-white font-bold'>{destination?.name}?</span><br />Note: you must also uninstall the Infra Connector from this cluster.</>}
        />
      </section>
    </div>
  )
}

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
  Cell: () => <span className='text-gray-400'>Cluster</span>
}]

export default function Destinations () {
  const { data: { items: destinations } = {}, error } = useSWR('/api/destinations')
  const { admin, loading: adminLoading } = useAdmin()
  const [selectedDestination, setSelectedDestination] = useState(null)
  const table = useTable({ columns, data: destinations || [] })

  const loading = adminLoading || (!destinations && !error)

  return (
    <>
      <Head>
        <title>Infrastructure - Infra</title>
      </Head>
      {!loading && (
        <div className='flex-1 flex h-full'>
          <main className='flex-1 flex flex-col space-y-4'>
            <PageHeader header='Infrastructure' buttonHref={admin && '/destinations/add'} buttonLabel='Infrastructure' />
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              : (
                <div>
                  <Table
                    {...table}
                    getRowProps={row => ({
                      onClick: () => setSelectedDestination(row.original),
                      style: {
                        cursor: 'pointer'
                      }
                    })}
                  />
                  {destinations?.length === 0 &&
                    <EmptyTable
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
              <SidebarContent destination={selectedDestination} admin={admin} setSelectedDestination={setSelectedDestination} />
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
