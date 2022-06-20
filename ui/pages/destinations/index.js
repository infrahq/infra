import useSWR, { useSWRConfig } from 'swr'
import Head from 'next/head'
import { useState } from 'react'
import dayjs from 'dayjs'
import { PlusSmIcon, MinusSmIcon } from '@heroicons/react/outline'

import { useAdmin } from '../../lib/admin'
import Dashboard from '../../components/layouts/dashboard'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import DeleteModal from '../../components/modals/delete'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'
import Grant from '../../components/grant'

function Details ({ destination, onDelete }) {
  const { data: auth } = useSWR('/api/users/self')
  const { admin } = useAdmin()
  const { data: { items: grants } = {} } = useSWR(() => `/api/grants?resource=${destination.resource}`)

  const { mutate } = useSWRConfig()
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  return (
    <div className='flex-1 flex flex-col space-y-6'>
      {grants?.filter(g => g.user === auth.id)?.length > 0 && (
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Connect</h3>
          <p className='text-2xs my-4'>Connect to this {destination.kind} via the <a target='_blank' href='https://infrahq.com/docs/install/install-infra-cli' className='underline text-violet-200 font-medium' rel='noreferrer'>Infra CLI</a></p>
          <pre className='px-4 py-3 rounded-md text-gray-300 bg-gray-900 text-2xs leading-normal overflow-auto'>
            infra login {window.location.host}<br />
            infra use {destination.resource}<br />
            kubectl get pods
          </pre>
        </section>
      )}
      {admin &&
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Access</h3>
          <Grant resource={destination.resource} />
        </section>}
      {destination.id && (
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Metadata</h3>
          <div className='pt-3 flex flex-col space-y-2'>
            <div className='flex flex-row items-center'>
              <div className='text-gray-400 text-2xs w-1/3'>ID</div>
              <div className='text-2xs font-mono'>{destination.id || '-'}</div>
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
      )}
      {admin && destination.id &&
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
              onDelete()
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
  Cell: ({ row, value }) => {
    return (
      <div className='flex py-2 items-center'>
        {row.canExpand && (
          <span {...row.getToggleRowExpandedProps({
            onClick: e => {
              row.toggleRowExpanded(!row.isExpanded)
              e.preventDefault()
              e.stopPropagation()
            },
            className: 'mr-3 w-6'
          })}
          >
            <div className={`bg-gray-900 ${row.isExpanded ? 'bg-gray-800' : 'bg-gray-900'} rounded-md flex items-center tracking-tight text-sm w-6 h-6`}>
              {row.isExpanded
                ? <MinusSmIcon className='w-4 h-4 m-auto' />
                : <PlusSmIcon className='w-4 h-4 m-auto' />}
            </div>
          </span>
        )}
        <span {...row.getToggleRowExpandedProps({ style: { marginLeft: `${row.depth * 36}px` } })}>
          {value}
        </span>
      </div>
    )
  }
}, {
  Header: 'Kind',
  accessor: v => v,
  width: '25%',
  Cell: ({ value }) => <span className='text-gray-400 px-2 py-0.5 bg-gray-800 rounded'>{value.kind}</span>
}]

export default function Destinations () {
  const { data: { items: destinations } = {}, error } = useSWR('/api/destinations')
  const { admin, loading: adminLoading } = useAdmin()
  const [selected, setSelected] = useState(null)

  const data = destinations
    ?.sort((a, b) => b?.created?.localeCompare(a.created))
    ?.map(d => ({
      ...d,
      kind: 'cluster',
      resource: d.name,
      subRows: d.resources?.map(r => ({
        name: r,
        resource: `${d.name}.${r}`,
        kind: 'namespace'
      }))
    })) || []

  const loading = adminLoading || (!destinations && !error)

  return (
    <>
      <Head>
        <title>Clusters - Infra</title>
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
                    columns={columns}
                    data={data}
                    getRowProps={row => ({
                      onClick: () => {
                        setSelected(row.original)
                        row.toggleRowExpanded(true)
                      },
                      className: selected?.resource === row.original.resource ? 'bg-gray-900/50' : 'cursor-pointer'
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
          {selected &&
            <Sidebar
              handleClose={() => setSelected(null)}
              title={selected.resource}
              iconPath='/destinations.svg'
            >
              <Details destination={selected} onDelete={() => setSelected(null)} />
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
