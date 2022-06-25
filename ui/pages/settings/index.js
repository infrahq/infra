import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'

import { sortBySubject } from '../../lib/grants'
import { useAdmin } from '../../lib/admin'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/modals/delete'
import Notification from '../../components/notification'
import GrantForm from '../../components/grant-form'

function AdminGrant ({ name, showRemove, onRemove }) {
  const [open, setOpen] = useState(false)

  return (
    <div className='flex justify-between items-center text-2xs group py-1'>
      <div className='py-1.5'>{name}</div>
      {showRemove && (
        <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
          <button onClick={() => setOpen(true)} className='flex-none px-2 py-1 -mr-2 cursor-pointer text-2xs text-gray-500 hover:text-violet-100'>Revoke</button>
          <DeleteModal
            open={open}
            setOpen={setOpen}
            primaryButtonText='Revoke'
            onSubmit={onRemove}
            title='Revoke Admin'
            message={(<>Are you sure you want to revoke admin access for <span className='font-bold text-white'>{name}</span>?</>)}
          />
        </div>
      )}
    </div>
  )
}

export default function Settings () {
  const router = useRouter()
  const { data: auth } = useSWR('/api/users/self')
  const { admin } = useAdmin()

  const { resetPassword } = router.query
  const [showNotification, setshowNotification] = useState(resetPassword === 'success')

  const { data: { items: users } = {} } = useSWR('/api/users')
  const { data: { items: groups } = {} } = useSWR('/api/groups')
  const { data: { items: grants } = {}, mutate } = useSWR('/api/grants?resource=infra&privilege=admin')

  const hasInfraProvider = auth?.providerNames.includes('infra')

  return (
    <>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      {auth && (
        <div className='flex-1 flex flex-col space-y-8 mt-6 mb-4'>
          <h1 className='text-xs mb-6 font-bold'>Settings</h1>
          {hasInfraProvider && (
            <div className='w-full max-w-md pb-12'>
              <div className='text-2xs leading-none uppercase text-gray-400 border-b border-gray-800 pb-3'>Account</div>
              <div className='pt-6 flex flex-col space-y-2'>
                <div className='flex group'>
                  <div className='flex flex-1 items-center'>
                    <div className='text-gray-400 text-2xs w-[26%]'>Email</div>
                    <div className='text-2xs'>{auth?.name}</div>
                  </div>
                </div>
                <div className='flex group'>
                  <div className='flex flex-1 items-center'>
                    <div className='text-gray-400 text-2xs w-[30%]'>Password</div>
                    <div className='text-2xs'>********</div>
                  </div>
                  <div className='flex justify-end'>
                    <Link href='/settings/password-reset'>
                      <a className='flex-none p-2 -mr-2 cursor-pointer uppercase text-2xs text-gray-500 hover:text-violet-100'>Change</a>
                    </Link>
                  </div>
                </div>
              </div>
            </div>
          )}
          {resetPassword && <Notification show={showNotification} setShow={setshowNotification} text='Password Successfully Reset' />}
        </div>
      )}
      {admin && (
        <div className='max-w-md'>
          <div className='text-2xs leading-none uppercase text-gray-400 border-b border-gray-800 pb-6'>Admins</div>
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
                body: JSON.stringify({ user, group, privilege: 'admin', resource: 'infra' })
              })

              mutate({ items: [...grants, await res.json()] })
            }}
          />
          <div className='mt-6'>
            {grants?.sort(sortBySubject)?.map(g => (
              <AdminGrant
                key={g.id}
                name={users?.find(u => g.user === u.id)?.name || groups?.find(group => g.group === group.id)?.name || ''}
                showRemove={g?.user !== auth?.id}
                onRemove={async () => {
                  await fetch(`/api/grants/${g.id}`, { method: 'DELETE' })
                  mutate({ items: grants.filter(x => x.id !== g.id) })
                }}
              />
            ))}
          </div>
        </div>
      )}
    </>
  )
}

Settings.layout = function (page) {
  return (
    <Dashboard>
      {page}
    </Dashboard>
  )
}
