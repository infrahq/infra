import useSWR from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import { useRouter } from 'next/router'
import Link from 'next/link'
import { TrashIcon } from '@heroicons/react/outline'
import moment from 'moment'

import { useUser } from '../../lib/hooks'
import { sortBySubject } from '../../lib/grants'

import GrantForm from '../../components/grant-form'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'
import Table from '../../components/table'

const TAB_ACCESS_KEY = { name: 'access_key', title: 'Access Key' }
const TAB_ORG_ADMINS = { name: 'admins', title: 'Organization Admins' }

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

  const { user, isAdmin } = useUser()

  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 20
  const {
    data: { items: accessKeys, totalPages, totalCount } = {},
    mutate: accessKeyMutate,
  } = useSWR(`/api/access-keys?userID=${user.id}&page=${page}&limit=${limit}`)

  const { data: { items: users } = {} } = useSWR('/api/users?limit=999')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=999')
  const { data: { items: grants } = {}, mutate } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=999'
  )
  const { data: { items: selfGroups } = {} } = useSWR(
    () => `/api/groups?userID=${user?.id}&limit=999`
  )

  const tabs = isAdmin ? [TAB_ACCESS_KEY, TAB_ORG_ADMINS] : []
  const tab = router.query.tab || TAB_ACCESS_KEY.name

  return (
    <div className='my-6'>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-1 flex-col'>
        {/* Header */}
        <h1 className='mb-6 font-display text-xl font-medium'>Settings</h1>

        {/* Tabs */}
        <div className='mb-3 border-b border-gray-200'>
          <nav className='-mb-px flex' aria-label='Tabs'>
            {tabs.map(t => (
              <Link
                key={t.name}
                href={{
                  pathname: `/settings/`,
                  query: { tab: t.name },
                }}
              >
                <a
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
                </a>
              </Link>
            ))}
          </nav>
        </div>

        {/* Access Key */}
        {tab === TAB_ACCESS_KEY.name && (
          <>
            <header className='my-2 flex items-center justify-end'>
              {/* Add dialog */}
              <Link href='/settings/add/access-key'>
                <a className='inline-flex items-center rounded-md border border-transparent bg-black  px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'>
                  Add access key
                </a>
              </Link>
            </header>
            <div className='mt-3 flex min-h-0 flex-1 flex-col'>
              <Table
                count={totalCount}
                pageCount={totalPages}
                pageIndex={parseInt(page) - 1}
                pageSize={limit}
                data={accessKeys}
                empty='No access keys'
                onPageChange={({ pageIndex }) => {
                  router.push({
                    pathname: router.pathname,
                    query: { ...router.query, p: pageIndex + 1 },
                  })
                }}
                columns={[
                  {
                    cell: function Cell(info) {
                      return (
                        <div className='flex flex-col py-0.5'>
                          <div className='truncate text-sm font-medium text-gray-700'>
                            {info.getValue()}
                          </div>
                          <div className='space-y-1 pt-2 text-3xs text-gray-500 sm:hidden'>
                            <div>
                              The key must be used before{' '}
                              <span className='font-semibold text-gray-700'>
                                {moment(
                                  info.row.original.extensionDeadline
                                ).format('YYYY/MM/DD')}
                              </span>
                            </div>
                            <div>
                              and will expire on{' '}
                              <span className='font-semibold text-gray-700'>
                                {moment(info.row.original.expires).format(
                                  'YYYY/MM/DD'
                                )}
                              </span>
                            </div>
                          </div>
                        </div>
                      )
                    },
                    header: () => <span>Name</span>,
                    accessorKey: 'name',
                  },
                  {
                    cell: info => (
                      <div className='hidden sm:table-cell'>
                        {info.getValue() ? moment(info.getValue()).from() : '-'}
                      </div>
                    ),
                    header: () => (
                      <span className='hidden sm:table-cell'>Created</span>
                    ),
                    accessorKey: 'created',
                  },
                  {
                    cell: info => (
                      <div className='hidden sm:table-cell'>
                        {info.getValue() ? moment(info.getValue()).from() : '-'}
                      </div>
                    ),
                    header: () => (
                      <span className='hidden sm:table-cell'>Expires</span>
                    ),
                    accessorKey: 'expires',
                  },
                  {
                    id: 'delete',
                    cell: function Cell(info) {
                      const [openDeleteModal, setOpenDeleteModal] =
                        useState(false)

                      const { name, id } = info.row.original

                      return (
                        <div className='flex justify-end'>
                          <button
                            type='button'
                            onClick={() => {
                              setOpenDeleteModal(true)
                            }}
                            className='group flex w-full items-center rounded-md bg-white px-2 py-1.5 text-xs font-medium text-red-500'
                          >
                            <TrashIcon className='mr-2 h-3.5 w-3.5' />
                            <span className='hidden sm:block'>Remove</span>
                          </button>
                          <DeleteModal
                            open={openDeleteModal}
                            setOpen={setOpenDeleteModal}
                            primaryButtonText='Remove'
                            onSubmit={async () => {
                              await fetch(`/api/access-keys/${id}`, {
                                method: 'DELETE',
                              })
                              setOpenDeleteModal(false)

                              accessKeyMutate()
                            }}
                            title='Remove Access Key'
                            message={
                              <div>
                                Are you sure you want to remove access key:{' '}
                                <span className='break-all font-bold'>
                                  {name}
                                </span>
                                ?
                              </div>
                            }
                          />
                        </div>
                      )
                    },
                  },
                ]}
              />
            </div>
          </>
        )}

        {/* Infra admins */}
        {tab === TAB_ORG_ADMINS.name && (
          <>
            <p className='mt-1 mb-4 text-xs text-gray-500'>
              These users and groups have full access to this organization.
            </p>
            <div className='mb-5 w-full rounded-lg border border-gray-200/75 px-5 py-3'>
              <GrantForm
                resource='infra'
                roles={['admin']}
                grants={grants}
                multiselect={false}
                onSubmit={async ({ user, group }) => {
                  // don't add grants that already exist
                  if (grants?.find(g => g.user === user && g.group === group)) {
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
  )
}
Settings.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
