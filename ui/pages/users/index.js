import Head from 'next/head'
import { useRouter } from 'next/router'
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
import EmptyData from '../../components/empty-data'
import Metadata from '../../components/metadata'
import RoleSelect from '../../components/role-select'
import RemoveButton from '../../components/remove-button'
import Pagination from '../../components/pagination'
import DeleteModal from '../../components/delete-modal'

const columns = [
  {
    Header: 'Name',
    width: '50%',
    accessor: u => u,
    Cell: ({ value: user }) => (
      <div className='flex items-center py-1.5'>
        <div className='flex h-7 w-7 select-none items-center justify-center rounded-md border border-gray-800'>
          <span className='text-3xs font-normal leading-none text-gray-400'>
            {user.name[0]}
          </span>
        </div>
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

  const { data: { items } = {}, mutate } = useSWR(
    `/api/grants?user=${id}&showInherited=1`
  )
  const { data: { items: groups } = {}, mutate: mutateGroups } = useSWR(
    `/api/groups?userID=${id}`
  )

  const { data: { items: infraAdmins } = {} } = useSWR(
    '/api/grants?resource=infra&privilege=admin'
  )

  const [open, setOpen] = useState(false)

  const grants = items?.filter(g => g.resource !== 'infra')
  const adminGroups = infraAdmins?.map(admin => admin.group)
  const metadata = [
    { title: 'ID', data: user?.id },
    {
      title: 'Created',
      data: user?.created ? dayjs(user.created).fromNow() : '-',
    },
    { title: 'Providers', data: user?.providerNames.join(', ') },
  ]

  const loading = [auth, grants, groups].some(x => !x)

  const handleRemoveGroupFromUser = async groupId => {
    const usersToRemove = [id]
    await fetch(`/api/groups/${groupId}/users`, {
      method: 'PATCH',
      body: JSON.stringify({ usersToRemove }),
    })
    mutateGroups({
      items: groups.filter(i => i.id !== groupId),
    })
  }

  return (
    !loading && (
      <div className='flex flex-1 flex-col space-y-6'>
        {admin && (
          <>
            <section>
              <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
                Access
              </h3>
              {grants
                ?.sort(sortByResource)
                ?.sort((a, b) => {
                  if (a.user === user.id) {
                    return -1
                  }

                  if (b.user === user.id) {
                    return 1
                  }

                  return 0
                })
                .map(g => (
                  <div
                    key={g.id}
                    className='flex items-center justify-between text-2xs'
                  >
                    <div>{g.resource}</div>
                    {g.user !== user.id ? (
                      <div className='flex'>
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
                    ) : (
                      <RoleSelect
                        role={g.privilege}
                        resource={g.resource}
                        remove
                        direction='left'
                        onRemove={async () => {
                          await fetch(`/api/grants/${g.id}`, {
                            method: 'DELETE',
                          })
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
                          await fetch(`/api/grants/${g.id}`, {
                            method: 'DELETE',
                          })
                          mutate({
                            items: [
                              ...grants.filter(f => f.id !== g.id),
                              await res.json(),
                            ],
                          })
                        }}
                      />
                    )}
                  </div>
                ))}
              {!grants?.length && !loading && (
                <EmptyData>
                  <div className='mt-6'>No access</div>
                </EmptyData>
              )}
            </section>
            <section>
              <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
                Groups
              </h3>
              <div className='mt-4'>
                {groups?.length === 0 && (
                  <EmptyData>
                    <div className='mt-6'>No groups</div>
                  </EmptyData>
                )}
                {groups.map(group => {
                  return (
                    <div
                      key={group.id}
                      className='group flex items-center justify-between truncate text-2xs'
                    >
                      <div className='py-2'>{group.name}</div>

                      <div className='flex justify-end text-right opacity-0 group-hover:opacity-100'>
                        <button
                          onClick={() =>
                            auth?.id === id && adminGroups?.includes(group.id)
                              ? setOpen(true)
                              : handleRemoveGroupFromUser(group.id)
                          }
                          className='-mr-2 flex-none cursor-pointer px-2 py-1 text-2xs text-gray-500 hover:text-violet-100'
                        >
                          Remove
                        </button>
                        <DeleteModal
                          open={open}
                          setOpen={setOpen}
                          primaryButtonText='Remove'
                          onSubmit={() => handleRemoveGroupFromUser(group.id)}
                          title='Remove Group'
                          message='Are you sure you want to remove yourself from this group? You will lose any access that this group grants.'
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            </section>
          </>
        )}
        <section>
          <h3 className='border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
            Metadata
          </h3>
          <Metadata data={metadata} />
        </section>
        <section className='flex flex-1 flex-col items-end justify-end py-6'>
          {auth.id !== id && (
            <RemoveButton
              onRemove={async () => {
                onDelete()
              }}
              modalTitle='Remove User'
              modalMessage={
                <>
                  Are you sure you want to remove{' '}
                  <span className='font-bold text-white'>{name}?</span>
                </>
              }
            />
          )}
        </section>
      </div>
    )
  )
}

export default function Users() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 13
  const {
    data: { items, totalPages, totalCount } = {},
    error,
    mutate,
  } = useSWR(`/api/users?page=${page}&limit=${limit}`)
  const { admin, loading: adminLoading } = useAdmin()
  const users = items?.filter(u => u.name !== 'connector')
  const table = useTable({
    columns,
    data: users || [],
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
              <div className='flex min-h-0 flex-1 flex-col overflow-y-auto px-6'>
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
                {users?.length === 0 && page === 1 && (
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
            {totalPages > 1 && (
              <Pagination
                curr={page}
                totalPages={totalPages}
                totalCount={totalCount}
                limit={limit}
              ></Pagination>
            )}
          </div>
          {selected && (
            <Sidebar
              onClose={() => setSelected(null)}
              title={selected.name}
              iconText={selected.name[0]}
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
