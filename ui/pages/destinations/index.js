import { useState } from 'react'
import useSWR from 'swr'
import Head from 'next/head'
import dayjs from 'dayjs'
import { PlusSmIcon, MinusSmIcon } from '@heroicons/react/outline'

import { sortBySubject, sortByPrivilege } from '../../lib/grants'
import { useAdmin } from '../../lib/admin'
import Dashboard from '../../components/layouts/dashboard'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import DeleteModal from '../../components/modals/delete'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'
import RoleSelect from '../../components/role-select'
import GrantForm from '../../components/grant-form'

function parent (resource = '') {
  const parts = resource.split('.')
  return parts.length > 1 ? parts[0] : null
}

function Details ({ destination, onDelete }) {
  const { resource } = destination

  const { admin } = useAdmin()
  const { data: auth } = useSWR('/api/users/self')
  const { data: { items: users } = {} } = useSWR('/api/users')
  const { data: { items: groups } = {} } = useSWR('/api/groups')
  const { data: { items: usergroups } = {} } = useSWR(() => auth ? `/api/groups?userID=${auth.id}` : null)
  const { data: { items: grants } = {}, mutate } = useSWR(`/api/grants?resource=${resource}`)
  const { data: { items: inherited } = {} } = useSWR(() => parent(resource) ? `/api/grants?resource=${parent(resource)}` : null)

  const connectable = grants?.find(g => g.user === auth?.id || usergroups.some(ug => ug.id === g.group))
  const empty = grants?.length === 0 && (parent(resource) ? inherited?.length === 0 : true)

  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  return (
    <div className='flex-1 flex flex-col space-y-6'>
      {admin &&
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Access</h3>
          <GrantForm
            roles={destination.roles}
            onSubmit={async ({ user, group, privilege }) => {
              // don't add grants that already exist
              if (grants?.find(g => g.user === user && g.group === group && g.privilege === privilege)) {
                return false
              }

              const res = await fetch('/api/grants', {
                method: 'POST',
                body: JSON.stringify({ user, group, privilege, resource })
              })

              mutate({ items: [...grants, await res.json()] })
            }}
          />
          <div className='mt-4'>
            {empty && (<div className='text-2xs text-gray-400 mt-6 italic'>No access</div>)}
            {grants?.sort(sortByPrivilege)?.sort(sortBySubject)?.map(g => (
              <div key={g.id} className='flex justify-between items-center text-2xs'>
                <div className='truncate'>
                  {users?.find(u => u.id === g.user)?.name}
                  {groups?.find(group => group.id === g.group)?.name}
                </div>
                <RoleSelect
                  role={g.privilege}
                  roles={destination.roles}
                  remove
                  onRemove={async () => {
                    await fetch(`/api/grants/${g.id}`, { method: 'DELETE' })
                    mutate({ items: grants.filter(x => x.id !== g.id) })
                  }}
                  onChange={async privilege => {
                    const res = await fetch('/api/grants', {
                      method: 'POST',
                      body: JSON.stringify({
                        ...g,
                        privilege
                      })
                    })

                    // delete old grant
                    await fetch(`/api/grants/${g.id}`, { method: 'DELETE' })

                    mutate({ items: [...grants.filter(f => f.id !== g.id), await res.json()] })
                  }}
                  direction='left'
                />
              </div>
            ))}
            {inherited?.sort(sortByPrivilege)?.sort(sortBySubject)?.map(g => (
              <div key={g.id} className='flex justify-between items-center text-2xs'>
                <div className='truncate'>
                  {users?.find(u => u.id === g.user)?.name}
                  {groups?.find(group => group.id === g.group)?.name}
                </div>
                <div className='flex-none flex'>
                  <div
                    title='This access is inherited by a parent resource and cannot be edited here'
                    className='relative pt-px mx-1 self-center text-2xs text-gray-400 border rounded px-2 bg-gray-800 border-gray-800'
                  >
                    inherited
                  </div>
                  <div className='relative flex-none pl-3 pr-8 w-32 py-2 text-left text-2xs text-gray-400'>
                    {g.privilege}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>}
      {connectable && (
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Connect</h3>
          <p className='text-2xs my-4'>Connect to this {destination?.kind || 'resource'} via the <a target='_blank' href='https://infrahq.com/docs/install/install-infra-cli' className='underline text-violet-200 font-medium' rel='noreferrer'>Infra CLI</a></p>
          <pre className='px-4 py-3 rounded-md text-gray-300 bg-gray-900 text-2xs leading-normal overflow-auto'>
            infra login {window.location.host}<br />
            infra use {destination.resource}<br />
            kubectl get pods
          </pre>
        </section>
      )}
      <section>
        <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Metadata</h3>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>ID</div>
            <div className='text-2xs'>{destination.id || '-'}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Kind</div>
            <div className='text-2xs'>{destination.kind || '-'}</div>
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
        <span {...row.getToggleRowExpandedProps()} className={`flex items-center ${row.depth === 0 ? 'h-6' : ''} ${row.canExpand ? '' : 'pl-9'}`}>
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
  const { data: { items: destinations } = {}, error, mutate } = useSWR('/api/destinations')
  const { admin, loading: adminLoading } = useAdmin()
  const [selected, setSelected] = useState(null)

  const data = destinations
    ?.sort((a, b) => b?.created?.localeCompare(a.created))
    ?.map(d => ({
      ...d,
      kind: 'cluster',
      resource: d.name,

      // Create "fake" destinations as subrows from resources
      subRows: d.resources?.map(r => ({
        name: r,
        resource: `${d.name}.${r}`,
        kind: 'namespace',
        roles: d.roles?.filter(r => r !== 'cluster-admin')
      }))
    })) || []

  const loading = adminLoading || !destinations

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
                <div className='flex flex-col flex-1 mx-6 min-h-0 overflow-y-scroll'>
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
              <Details
                destination={selected} onDelete={() => {
                  fetch(`/api/destinations/${selected.id}`, { method: 'DELETE' })
                  mutate(destinations.filter(d => d?.id !== selected.id))
                  setSelected(null)
                }}
              />
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
