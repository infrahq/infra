import useSWR from 'swr'
import { useState } from 'react'
import Head from 'next/head'

import { sortBySubject } from '../../lib/grants'

import GrantForm from '../../components/grant-form'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'

function AdminList({ grants, users, groups, onRemove, auth, selfGroups }) {
  const [open, setOpen] = useState(false)
  const [deleteId, setDeleteId] = useState(null)

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
    <div className='-my-2 -mx-4 overflow-x-auto sm:-mx-6 lg:-mx-8'>
      <div className='inline-block min-w-full py-2 px-4 align-middle md:px-6 lg:px-8'>
        <table className='min-w-full divide-y divide-gray-200 border-t border-gray-200'>
          <tbody>
            {grantsList?.map(grant => (
              <tr key={grant.id} className='border-b border-gray-200'>
                <td className='whitespace-nowrap py-4'>
                  <div className='truncate text-sm font-medium text-gray-900'>
                    {grant.name}
                  </div>
                </td>
                <td className='py-4 text-right'>
                  <button
                    onClick={() => {
                      setDeleteId(grant.id)
                      setOpen(true)
                    }}
                    className='cursor-pointer pr-4 text-xs text-blue-600 hover:text-blue-900'
                  >
                    Revoke
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
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
            !grantsList?.find(grant => grant.id === deleteId)?.message ? (
              <>
                Are you sure you want to revoke admin access for{' '}
                <span className='font-bold'>
                  {grantsList?.find(grant => grant.id === deleteId)?.name}
                </span>
                ?
              </>
            ) : (
              grantsList?.find(grant => grant.id === deleteId)?.message
            )
          }
        />
      </div>
    </div>
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
        <h1 className='mb-6 text-xl font-medium'>Settings</h1>

        {/* Infra admins */}
        <h2 className='text-md font-medium'>Organization Admins</h2>
        <p className='mt-1 text-xs text-gray-500'>
          These users have full access to this organizations and its clusters.
        </p>
        <div className='flex flex-col space-y-3'>
          <GrantForm
            resource='infra'
            roles={['admin']}
            onSubmit={async ({ user, group }) => {
              // don't add grants that already exist
              if (grants?.find(g => g.user === user && g.group === group)) {
                return false
              }

              const res = await fetch('/api/grants', {
                method: 'POST',
                body: JSON.stringify({
                  user,
                  group,
                  privilege: 'admin',
                  resource: 'infra',
                }),
              })

              mutate({ items: [...grants, await res.json()] })
            }}
          />
          <div>
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
      </div>
    </div>
  )
}
Settings.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
