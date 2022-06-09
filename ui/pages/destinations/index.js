import useSWR, { useSWRConfig } from 'swr'
import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import dayjs from 'dayjs'
import { PlusSmIcon, MinusSmIcon } from '@heroicons/react/outline'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import DeleteModal from '../../components/modals/delete'
import Grant from '../../components/grant'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'
import NamespaceGrant from '../../components/namespace-grant'

function SidebarNamespaceContent ({ namespace }) {
  return (
    <div className='flex-1 flex flex-col space-y-6'>
      <section>
        <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Access</h3>
        <NamespaceGrant destinationId={namespace.destinationId} namespaceName={namespace.name} />
      </section>
    </div>
  )
}

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
          <pre className='px-4 py-3 rounded-md text-gray-300 bg-gray-900 text-2xs leading-normal overflow-auto'>
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
        <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Metadata</h3>
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
            <div className='text-2xs'>{destination?.updated ? dayjs(destination.updated).fromNow() : '-'}</div>
          </div>
        </div>
      </section>
      {admin &&
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
        </section>}
    </div>
  )
}

const columns = [{
  Header: 'Name',
  accessor: 'name',
  id: 'expander',
  Cell: ({ row, value }) => {
    return (
      <div className='flex py-3 items-center'>
        <span className='mr-3' {...row.getToggleRowExpandedProps()}>
          <div className='border border-gray-900 bg-gray-900 rounded-md flex items-center tracking-tight text-sm w-6 h-6'>
            {row.isExpanded ? <MinusSmIcon className='w-4 h-4 m-auto' /> : <PlusSmIcon className='w-4 h-4 m-auto' />}
          </div>
        </span>
        {value}
      </div>
    )
  }
}, {
  Header: 'Kind',
  Cell: () => <span className='text-gray-400'>cluster</span>
}]

export default function Destinations () {
  const { data: { items: destinations } = {}, error } = useSWR('/api/destinations')
  const { admin, loading: adminLoading } = useAdmin()
  const [selectedDestination, setSelectedDestination] = useState(null)
  const [selectedNamespace, setSelectedNamespace] = useState(null)

  const table = useTable({ columns, data: destinations?.sort((a, b) => b.created?.localeCompare(a.created)) || [] })

  const loading = adminLoading || (!destinations && !error)

  const selectDestination = (row) => {
    setSelectedDestination(row)
    setSelectedNamespace(null)
  }

  const selectNamespace = (row) => {
    setSelectedNamespace(row)
    setSelectedDestination(null)
  }

  const handleClose = () => {
    setSelectedDestination(null)
    setSelectedNamespace(null)
  }

  function renderRowSubComponent (row) {
    const { name: destination, id: destinationId, resources, roles } = row.original
    const rowSubData = resources.map(resource => {
      return {
        destination,
        destinationId,
        name: resource,
        roles: roles.filter(role => role !== 'cluster-admin')
      }
    })

    const subColumns = [{
      Header: 'Namespaces',
      accessor: 'name',
      Cell: ({ value }) => {
        return (
          <div className='flex py-3 items-center'>
            {value}
          </div>
        )
      }
    }, {
      Header: 'Kind',
      Cell: () => <span className='text-gray-400'>namespace</span>
    }]

    return (
      <div className='ml-16 mt-6 mb-6'>
        <Table
          subTable
          columns={subColumns}
          data={rowSubData}
          getRowProps={row => ({
            onClick: () => selectNamespace(row.original),
            style: {
              cursor: 'pointer',
              background: row.original.name === selectedNamespace?.name ? '#151A1E' : ''
            }
          })}
        />
      </div>
    )
  }

  return (
    <>
      <Head>
        <title>Infrastructure - Infra</title>
      </Head>
      {!loading && (
        <div className='flex-1 flex h-full'>
          <div className='flex-1 flex flex-col space-y-4'>
            <PageHeader header='Clusters' buttonHref={admin && '/destinations/add'} buttonLabel='Cluster' />
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              : (
                <div className='flex flex-col flex-1 px-6 min-h-0 overflow-y-scroll'>
                  <Table
                    {...table}
                    renderRowSubComponent={renderRowSubComponent}
                    getRowProps={row => ({
                      onClick: () => selectDestination(row.original),
                      style: {
                        cursor: 'pointer',
                        background: row.original.id === selectedDestination?.id ? '#151A1E' : ''
                      }
                    })}
                  />
                  {destinations?.length === 0 &&
                    <EmptyTable
                      title='There are no clusters'
                      subtitle='There is currently no cluster connected to Infra'
                      iconPath='/destinations.svg'
                      buttonHref={admin && '/destinations/add'}
                      buttonText='Cluster'
                    />}
                </div>
                )}
          </div>
          {selectedDestination &&
            <Sidebar
              handleClose={() => handleClose()}
              title={selectedDestination.name}
              iconPath='/destinations.svg'
            >
              <SidebarContent destination={selectedDestination} admin={admin} setSelectedDestination={setSelectedDestination} />
            </Sidebar>}
          {selectedNamespace &&
            <Sidebar
              handleClose={() => handleClose()}
              title={`.../${selectedNamespace.name}`}
              iconPath='/destinations.svg'
            >
              <SidebarNamespaceContent namespace={selectedNamespace} />
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
