import useSWR from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import { useRouter } from 'next/router'
import Link from 'next/link'

import { useUser } from '../../lib/hooks'
import { sortBySubject } from '../../lib/grants'

import GrantForm from '../../components/grant-form'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'
import Table from '../../components/table'

const TAB_ORG_ADMINS = { name: 'admins', title: 'Organization Admins' }
const TAB_PROVIDERS = { name: 'providers', title: 'Providers' }

function AdminList({ grants, users, groups, onRemove, auth, selfGroups }) {
  const grantsList = grants?.sort(sortBySubject)?.map(grant => {
    const message =
      grant?.user === auth?.id
        ? 'Are you sure you want to remove yourself as an admin?'
        : selfGroups?.some(g => g.id === grant.group)
        ? `Are you sure you want to revoke this group's admin access? You are a member of this group.`
        : undefined

    const name =
      users?.find(u => grant.user === u.id)?.name ||
      groups?.find(group => grant.group === group.id)?.name ||
      ''

    return { ...grant, message, name }
  })

  return (
    <Table
      data={grantsList}
      columns={[
        {
          cell: function Cell(info) {
            return (
              <div className='flex flex-col'>
                <div className='flex items-center font-medium text-gray-700'>
                  {info.getValue()}
                </div>
                <div className='text-2xs text-gray-500'>
                  {info.row.original.user && 'User'}
                  {info.row.original.group && 'Group'}
                </div>
              </div>
            )
          },
          header: () => <span>Admin</span>,
          accessorKey: 'name',
        },
        {
          cell: function Cell(info) {
            const [open, setOpen] = useState(false)
            const [deleteId, setDeleteId] = useState(null)

            return (
              grants?.length > 1 && (
                <div className='text-right'>
                  <button
                    onClick={() => {
                      setDeleteId(info.row.original.id)
                      setOpen(true)
                    }}
                    className='p-1 text-2xs text-gray-500/75 hover:text-gray-600'
                  >
                    Revoke
                    <span className='sr-only'>{info.row.original.name}</span>
                  </button>
                  <DeleteModal
                    open={open}
                    setOpen={setOpen}
                    primaryButtonText='Revoke'
                    onSubmit={() => {
                      onRemove(deleteId)
                      setOpen(false)
                    }}
                    title='Revoke Admin'
                    message={
                      !grantsList?.find(grant => grant.id === deleteId)
                        ?.message ? (
                        <>
                          Are you sure you want to revoke admin access for{' '}
                          <span className='font-bold'>
                            {
                              grantsList?.find(grant => grant.id === deleteId)
                                ?.name
                            }
                          </span>
                          ?
                        </>
                      ) : (
                        grantsList?.find(grant => grant.id === deleteId)
                          ?.message
                      )
                    }
                  />
                </div>
              )
            )
          },
          id: 'delete',
        },
      ]}
    />
  )
}

export default function Settings() {
  const router = useRouter()
  const { user } = useUser()

  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 1000

  const { data: { items: providers } = {} } = useSWR(
    `/api/providers?page=${page}&limit=${limit}`
  )
  const { data: { items: users } = {} } = useSWR(`/api/users?limit=${limit}`)
  const { data: { items: groups } = {} } = useSWR(`/api/groups?limit=${limit}`)
  const { data: { items: grants } = {}, mutate } = useSWR(
    `/api/grants?resource=infra&privilege=admin&limit=${limit}`
  )
  const { data: { items: selfGroups } = {} } = useSWR(
    () => `/api/groups?userID=${user?.id}&limit=${limit}`
  )

  const tabs = [TAB_ORG_ADMINS, TAB_PROVIDERS]
  const tab = router.query.tab || TAB_ORG_ADMINS.name

  return (
    <div className='my-6'>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-1 flex-col'>
        {/* Header */}
        <h1 className='mb-6 font-display text-xl font-medium'>Settings</h1>

        {/* Tabs */}
        <div className='border-b border-gray-200'>
          <nav className='-mb-px flex space-x-4' aria-label='Tabs'>
            {tabs.map(t => (
              <Link
                key={t.name}
                href={{
                  pathname: `/settings/`,
                  query: { tab: t.name },
                }}
                className={`
                ${
                  tab === t.name
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-600'
                }
                 whitespace-nowrap border-b-2 py-2 px-5 text-sm font-medium capitalize transition-colors`}
                aria-current={tab.current ? 'page' : undefined}
              >
                {t.title}
              </Link>
            ))}
          </nav>
        </div>
        <div className='mt-3'>
          {tab === TAB_PROVIDERS.name && (
            <>
              <header className='my-2 flex items-center justify-end'>
                {/* Add dialog */}
                <Link
                  href='/settings/providers/add'
                  className='inline-flex items-center rounded-md border border-transparent bg-black  px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
                >
                  Connect provider
                </Link>
              </header>
              <div className='mt-3 flex min-h-0 flex-1 flex-col'>
                <Table
                  href={row => `/settings/providers/${row.original.id}`}
                  data={providers}
                  empty='No providers'
                  columns={[
                    {
                      cell: info => (
                        <div className='flex flex-row items-center py-1'>
                          <div className='mr-3 flex h-9 w-9 flex-none items-center justify-center rounded-md border border-gray-200'>
                            <img
                              alt='provider icon'
                              className='h-4'
                              src={`/providers/${info.row.original.kind}.svg`}
                            />
                          </div>
                          <div className='flex flex-col'>
                            <div className='text-sm font-medium text-gray-700'>
                              {info.getValue()}
                            </div>
                            <div className='text-2xs text-gray-500 sm:hidden'>
                              {info.row.original.url}
                            </div>
                            <div className='font-mono text-2xs text-gray-400 lg:hidden'>
                              {info.row.original.clientID}
                            </div>
                          </div>
                        </div>
                      ),
                      header: () => <span>Name</span>,
                      accessorKey: 'name',
                    },
                    {
                      cell: info => (
                        <div className='hidden sm:table-cell'>
                          {info.getValue()}
                        </div>
                      ),
                      header: () => (
                        <span className='hidden sm:table-cell'>URL</span>
                      ),
                      accessorKey: 'url',
                    },
                    {
                      cell: info => (
                        <div className='hidden font-mono lg:table-cell'>
                          {info.getValue()}
                        </div>
                      ),
                      header: () => (
                        <span className='hidden lg:table-cell'>Client ID</span>
                      ),
                      accessorKey: 'clientID',
                    },
                  ]}
                />
              </div>
            </>
          )}
          {/* Infra admins */}
          {tab === TAB_ORG_ADMINS.name && (
            <>
              <div className='mb-3 flex flex-col justify-between'>
                <p className='mt-1 mb-4 text-xs text-gray-500'>
                  These users have full access to this organization.
                </p>
                <div className='w-full rounded-lg border border-gray-200/75 px-5 py-3'>
                  <GrantForm
                    resource='infra'
                    roles={['admin']}
                    grants={grants}
                    multiselect={false}
                    onSubmit={async ({ user, group }) => {
                      // don't add grants that already exist
                      if (
                        grants?.find(g => g.user === user && g.group === group)
                      ) {
                        return false
                      }

                      await fetch('/api/grants', {
                        method: 'POST',
                        body: JSON.stringify({
                          user,
                          group,
                          privilege: 'admin',
                          resource: 'infra',
                        }),
                      })

                      // TODO: add optimistic updates
                      mutate()
                    }}
                  />
                </div>
              </div>
              <AdminList
                grants={grants}
                users={users}
                groups={groups}
                selfGroups={selfGroups}
                auth={user}
                onRemove={async grantId => {
                  await fetch(`/api/grants/${grantId}`, {
                    method: 'DELETE',
                  })
                  mutate({ items: grants?.filter(x => x.id !== grantId) })
                }}
              />
            </>
          )}
        </div>
      </div>
    </div>
  )
}
Settings.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
