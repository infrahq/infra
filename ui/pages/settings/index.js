import useSWR from 'swr'
import { useState } from 'react'
import Head from 'next/head'

import { sortBySubject } from '../../lib/grants'

import GrantForm from '../../components/grant-form'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'
import Table from '../../components/table'

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
  const { data: auth } = useSWR('/api/users/self')

  const { data: { items: users } = {} } = useSWR('/api/users?limit=999')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=999')
  const { data: { items: grants } = {}, mutate } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=999'
  )
  const { data: { items: selfGroups } = {} } = useSWR(
    `/api/groups?userID=${auth?.id}&limit=999`
  )

  return (
    <div className='my-6'>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-1 flex-col'>
        {/* Header */}
        <h1 className='mb-6 font-display text-xl font-medium'>Settings</h1>

        {/* Infra admins */}
        <div className='mb-3 flex flex-col justify-between'>
          <h2 className='font-display text-lg font-medium'>
            Organization Admins
          </h2>
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
        </div>
        <AdminList
          grants={grants}
          users={users}
          groups={groups}
          selfGroups={selfGroups}
          auth={auth}
          onRemove={async grantId => {
            await fetch(`/api/grants/${grantId}`, {
              method: 'DELETE',
            })
            mutate({ items: grants?.filter(x => x.id !== grantId) })
          }}
        />
      </div>
    </div>
  )
}
Settings.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
