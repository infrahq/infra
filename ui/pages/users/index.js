import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import useSWR from 'swr'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'
import { sortByResource } from '../../lib/grants'
import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/page-header'
import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import Sidebar from '../../components/sidebar'
import ProfileIcon from '../../components/profile-icon'
import DeleteModal from '../../components/delete-modal'
import RoleSelect from '../../components/role-select'

const columns = [
  {
    Header: 'Name',
    width: '50%',
    accessor: u => u,
    Cell: ({ value: user }) => (
      <div className='flex items-center py-1.5'>
        <ProfileIcon name={user.name[0]} />
        <div className='ml-3 flex min-w-0 flex-1 flex-col leading-tight'>
          <div className='truncate'>{user.name}</div>
        </div>
      </div>
    ),
  },
  {
    Header: 'Last Seen',
    width: '25%',
    accessor: u => u,
    Cell: ({ value: user }) => (
      <div className='text-name text-gray-400'>
        {user.lastSeenAt ? dayjs(user.lastSeenAt).fromNow() : '-'}
      </div>
    ),
  },
  {
    Header: 'Added',
    width: '25%',
    accessor: u => u,
    Cell: ({ value: user }) => (
      <div className='text-name text-gray-400'>
        {user?.created ? dayjs(user.created).fromNow() : '-'}
      </div>
    ),
  },
]

function Details({ user, admin, onDelete }) {
  const { id, name } = user
  const { data: auth } = useSWR('/api/users/self')

  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  const { data: { items } = {}, mutate } = useSWR(`/api/grants?user=${id}`)
  const { data: { items: groups } = {} } = useSWR(`/api/groups?userID=${id}`)
  const { data: groupGrantDatas } = useSWR(
    () => (groups ? groups.map(g => `/api/grants?group=${g.id}`) : null),
    (...urls) => Promise.all(urls.map(url => fetch(url).then(r => r.json())))
  )

  const grants = items?.filter(g => g.resource !== 'infra')
  const inherited = groupGrantDatas
    ?.map(g => g?.items || [])
    ?.flat()
    ?.filter(g => g.resource !== 'infra')

  const loading = [
    auth,
    grants,
    groups,
    groups?.length ? inherited : true,
  ].some(x => !x)

  return (
    <div className='flex flex-1 flex-col space-y-6'>
      {admin && (
        <section>
          <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
            Access
          </h3>
          {grants?.sort(sortByResource)?.map(g => (
            <div
              key={g.id}
              className='flex items-center justify-between text-2xs'
            >
              <div>{g.resource}</div>
              <RoleSelect
                role={g.privilege}
                resource={g.resource}
                remove
                direction='left'
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
              />
            </div>
          ))}
          {inherited?.sort(sortByResource)?.map(g => (
            <div
              key={g.id}
              className='flex items-center justify-between text-2xs'
            >
              <div>{g.resource}</div>
              <div className='flex flex-none'>
                <div
                  title='This access is inherited by a group and cannot be edited here'
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
          {!grants?.length && !inherited?.length && !loading && (
            <div className='mt-6 text-2xs italic text-gray-400'>No access</div>
          )}
        </section>
      )}
      <section>
        <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
          Metadata
        </h3>
        <div className='flex flex-col space-y-2 pt-3'>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>ID</div>
            <div className='text-2xs'>{user?.id}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='w-1/3 text-2xs text-gray-400'>Created</div>
            <div className='text-2xs'>
              {user?.created ? dayjs(user.created).fromNow() : '-'}
            </div>
          </div>
        </div>
      </section>
      <section className='flex flex-1 flex-col items-end justify-end py-6'>
        {auth.id !== id && (
          <button
            type='button'
            onClick={() => setDeleteModalOpen(true)}
            className='flex items-center rounded-md border border-violet-300 px-6 py-3 text-2xs text-violet-100'
          >
            Remove
          </button>
        )}
        <DeleteModal
          open={deleteModalOpen}
          setOpen={setDeleteModalOpen}
          onSubmit={async () => {
            setDeleteModalOpen(false)
            onDelete()
          }}
          title='Remove User'
          message={
            <>
              Are you sure you want to remove{' '}
              <span className='font-bold text-white'>{name}?</span>
            </>
          }
        />
      </section>
    </div>
  )
}

export default function Users() {
  const { data: { items } = {}, error, mutate } = useSWR('/api/users')
  const { admin, loading: adminLoading } = useAdmin()
  const users = items?.filter(u => u.name !== 'connector')
  const table = useTable({
    columns,
    data: users?.sort((a, b) => b.created?.localeCompare(a.created)) || [],
  })
  const [selected, setSelected] = useState(null)

  const loading = adminLoading || (!users && !error)

  return (
    <>
      <Head>
        <title>Users - Infra</title>
      </Head>
      {!loading && (
        <div className='flex h-full flex-1'>
          <div className='flex flex-1 flex-col space-y-4'>
            <PageHeader
              header='Users'
              buttonHref={admin && '/users/add'}
              buttonLabel='User'
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
                {users?.length === 0 && (
                  <EmptyTable
                    title='There are no users'
                    subtitle='Invite users to Infra and manage their access.'
                    iconPath='/users.svg'
                    buttonHref={admin && '/users/add'}
                    buttonText='Users'
                  />
                )}
              </div>
            )}
          </div>
          {selected && (
            <Sidebar
              onClose={() => setSelected(null)}
              title={selected.name}
              profileIcon={selected.name[0]}
            >
              <Details
                user={selected}
                admin={admin}
                onDelete={() => {
                  mutate(async ({ items: users } = { items: [] }) => {
                    await fetch(`/api/users/${selected.id}`, {
                      method: 'DELETE',
                    })

                    return {
                      items: users?.filter(u => u?.id !== selected.id),
                    }
                  })

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

Users.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
