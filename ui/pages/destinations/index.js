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
import DeleteModal from '../../components/delete-modal'
import PageHeader from '../../components/page-header'
import Sidebar from '../../components/sidebar'
import RoleSelect from '../../components/role-select'
import GrantForm from '../../components/grant-form'

function parent(resource = '') {
  const parts = resource.split('.')
  return parts.length > 1 ? parts[0] : null
}

function Details({ destination, onDelete }) {
  const { resource } = destination

  const { admin } = useAdmin()
  const { data: auth } = useSWR('/api/users/self')
  const { data: { items: users } = {} } = useSWR('/api/users')
  const { data: { items: groups } = {} } = useSWR('/api/groups')
  const { data: { items: usergroups } = {} } = useSWR(() =>
    auth ? `/api/groups?userID=${auth.id}` : null
  )
  const { data: { items: grants } = {}, mutate } = useSWR(
    `/api/grants?resource=${resource}`
  )
  const { data: { items: inherited } = {} } = useSWR(() =>
    parent(resource) ? `/api/grants?resource=${parent(resource)}` : null
  )

  const connectable = grants?.find(
    g => g.user === auth?.id || usergroups.some(ug => ug.id === g.group)
  )
  const empty =
    grants?.length === 0 && (parent(resource) ? inherited?.length === 0 : true)

  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  return (
    <div className='flex flex-1 flex-col space-y-6'>
      {admin && (
        <section>
          <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
            Access
          </h3>
          <GrantForm
            roles={destination.roles}
            onSubmit={async ({ user, group, privilege }) => {
              // don't add grants that already exist
              if (
                grants?.find(
                  g =>
                    g.user === user &&
                    g.group === group &&
                    g.privilege === privilege
                )
              ) {
                return false
              }

              const res = await fetch('/api/grants', {
                method: 'POST',
                body: JSON.stringify({ user, group, privilege, resource }),
              })

              mutate({ items: [...grants, await res.json()] })
            }}
          />
          <div className='mt-4'>
            {empty && (
              <div className='mt-6 text-2xs italic text-gray-400'>
                No access
              </div>
            )}
            {grants
              ?.sort(sortByPrivilege)
              ?.sort(sortBySubject)
              ?.map(g => (
                <div
                  key={g.id}
                  className='flex items-center justify-between text-2xs'
                >
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
                          privilege,
                        }),
                      })

                      // delete old grant
                      await fetch(`/api/grants/${g.id}`, { method: 'DELETE' })

                      mutate({
                        items: [
                          ...grants.filter(f => f.id !== g.id),
                          await res.json(),
                        ],
                      })
                    }}
                    direction='left'
                  />
                </div>
              ))}
            {inherited
              ?.sort(sortByPrivilege)
              ?.sort(sortBySubject)
              ?.map(g => (
                <div
                  key={g.id}
                  className='flex items-center justify-between text-2xs'
                >
                  <div className='truncate'>
                    {users?.find(u => u.id === g.user)?.name}
                    {groups?.find(group => group.id === g.group)?.name}
                  </div>
                  <div className='flex flex-none'>
                    <div
                      title='This access is inherited by a parent resource and cannot be edited here'
                      className='relative mx-1 self-center rounded border border-gray-800 bg-gray-800 px-2 pt-px text-2xs text-gray-400'
                    >
                      inherited
                    </div>
                    <div className='relative w-32 flex-none py-2 pl-3 pr-8 text-left text-2xs text-gray-400'>
                      {g.privilege}
                    </div>
                  </div>
                </div>
              ))}
          </div>
        </section>
      )}
      {connectable && (
        <section>
          <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
            Connect
          </h3>
          <p className='my-4 text-2xs'>
            Connect to this {destination?.kind || 'resource'} via the{' '}
            <a
              target='_blank'
              href='https://infrahq.com/docs/install/install-infra-cli'
              className='font-medium text-violet-200 underline'
              rel='noreferrer'
            >
              Infra CLI
            </a>
          </p>
          <pre className='overflow-auto rounded-md bg-gray-900 px-4 py-3 text-2xs leading-normal text-gray-300'>
            infra login {window.location.host}
            <br />
            infra use {destination.resource}
            <br />
            kubectl get pods
          </pre>
        </section>
      )}
      <section>
        <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
          Metadata
        </h3>
        <div className='flex flex-col space-y-2 pt-3'>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>ID</div>
            <div className='text-2xs'>{destination.id || '-'}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Kind</div>
            <div className='text-2xs'>{destination.kind || '-'}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Added</div>
            <div className='text-2xs'>
              {destination?.created
                ? dayjs(destination.created).fromNow()
                : '-'}
            </div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Updated</div>
            <div className='text-2xs'>
              {destination?.updated
                ? dayjs(destination.updated).fromNow()
                : '-'}
            </div>
          </div>
        </div>
      </section>
      {admin && destination.id && (
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
            onSubmit={async () => {
              setDeleteModalOpen(false)
              onDelete()
            }}
            title='Remove Cluster'
            message={
              <>
                Are you sure you want to disconnect{' '}
                <span className='font-bold text-white'>
                  {destination?.name}?
                </span>
                <br />
                Note: you must also uninstall the Infra Connector from this
                cluster.
              </>
            }
          />
        </section>
      )}
    </div>
  )
}

const columns = [
  {
    Header: 'Name',
    accessor: 'name',
    Cell: ({ row, value }) => {
      return (
        <div className='flex items-center py-2'>
          {row.canExpand && (
            <span
              {...row.getToggleRowExpandedProps({
                onClick: e => {
                  row.toggleRowExpanded(!row.isExpanded)
                  e.preventDefault()
                  e.stopPropagation()
                },
                className: 'mr-3 w-6',
              })}
            >
              <div
                className={`bg-gray-900 ${
                  row.isExpanded ? 'bg-gray-800' : 'bg-gray-900'
                } flex h-6 w-6 items-center rounded-md text-sm tracking-tight`}
              >
                {row.isExpanded ? (
                  <MinusSmIcon className='m-auto h-4 w-4' />
                ) : (
                  <PlusSmIcon className='m-auto h-4 w-4' />
                )}
              </div>
            </span>
          )}
          <span
            {...row.getToggleRowExpandedProps()}
            className={`flex items-center ${row.depth === 0 ? 'h-6' : ''} ${
              row.canExpand ? '' : 'pl-9'
            }`}
          >
            {value}
          </span>
        </div>
      )
    },
  },
  {
    Header: 'Kind',
    accessor: v => v,
    width: '25%',
    Cell: ({ value }) => (
      <span className='rounded bg-gray-800 px-2 py-0.5 text-gray-400'>
        {value.kind}
      </span>
    ),
  },
]

export default function Destinations() {
  const {
    data: { items: destinations } = {},
    error,
    mutate,
  } = useSWR('/api/destinations')
  const { admin, loading: adminLoading } = useAdmin()
  const [selected, setSelected] = useState(null)

  const data =
    destinations
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
          roles: d.roles?.filter(r => r !== 'cluster-admin'),
        })),
      })) || []

  const loading = adminLoading || !destinations

  return (
    <>
      <Head>
        <title>Clusters - Infra</title>
      </Head>
      {!loading && (
        <div className='flex h-full flex-1'>
          <div className='flex flex-1 flex-col space-y-4'>
            <PageHeader
              header='Clusters'
              buttonHref={admin && '/destinations/add'}
              buttonLabel='Cluster'
            />
            {error?.status ? (
              <div className='my-20 text-center text-sm font-light text-gray-300'>
                {error?.info?.message}
              </div>
            ) : (
              <div className='mx-6 flex min-h-0 flex-1 flex-col overflow-y-scroll'>
                <Table
                  columns={columns}
                  data={data}
                  getRowProps={row => ({
                    onClick: () => {
                      setSelected(row.original)
                      row.toggleRowExpanded(true)
                    },
                    className:
                      selected?.resource === row.original.resource
                        ? 'bg-gray-900/50'
                        : 'cursor-pointer',
                  })}
                />
                {destinations?.length === 0 && (
                  <EmptyTable
                    title='There are no clusters'
                    subtitle='There is currently no cluster connected to Infra'
                    iconPath='/destinations.svg'
                    buttonHref={admin && '/destinations/add'}
                    buttonText='Cluster'
                  />
                )}
              </div>
            )}
          </div>
          {selected && (
            <Sidebar
              handleClose={() => setSelected(null)}
              title={selected.resource}
              iconPath='/destinations.svg'
            >
              <Details
                destination={selected}
                onDelete={() => {
                  fetch(`/api/destinations/${selected.id}`, {
                    method: 'DELETE',
                  })
                  mutate(destinations.filter(d => d?.id !== selected.id))
                  setSelected(null)
                }}
              />
            </Sidebar>
          )}
        </div>
      )}
    </>
  )
}

Destinations.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
