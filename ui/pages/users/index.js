import Head from 'next/head'
import { useRouter } from 'next/router'
import useSWR from 'swr'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import EmptyTable from '../../components/empty-table'
import Dashboard from '../../components/layouts/dashboard'
import Pagination from '../../components/pagination'
import { UsersIcon } from '@heroicons/react/outline'
import PageHeader from '../../components/page-header'

function UserTable({ users }) {
  const router = useRouter()

  return (
    <div className='overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg'>
      <table className='min-w-full divide-y divide-gray-300'>
        <thead className='bg-gray-50'>
          <tr>
            <th
              scope='col'
              className='py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6'
            >
              Name
            </th>
            <th
              scope='col'
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 lg:table-cell'
            >
              Providers
            </th>
            <th
              scope='col'
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 sm:table-cell'
            >
              Last Seen
            </th>
            <th
              scope='col'
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 sm:table-cell'
            >
              Created
            </th>
          </tr>
        </thead>
        <tbody className='divide-y divide-gray-200 bg-white'>
          {users?.map(user => (
            <tr
              key={user.id}
              onClick={() => router.replace(`/users/${user.id}`)}
              className='hover:cursor-pointer hover:bg-gray-100'
            >
              <td className='w-full max-w-0 py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:w-auto sm:max-w-none sm:pl-6'>
                <div className='flex items-center py-1.5'>
                  <div className='text-sm sm:max-w-[10rem]'>{user.name}</div>
                </div>
                <dl className='font-normal lg:hidden'>
                  <dt className='sr-only sm:hidden'>Providers</dt>
                  <dd className='mt-1 text-gray-700'>
                    {user.providerNames?.map((provider, index) => (
                      <span key={provider}>
                        {provider}
                        {index !== user.providerNames.length - 1 && <>, </>}
                      </span>
                    ))}
                  </dd>
                  <dt className='sr-only sm:hidden'>Last Seen</dt>
                  <dd className='mt-1 truncate text-gray-700 sm:hidden'>
                    {user?.lastSeenAt ? (
                      <>{dayjs(user.lastSeenAt).fromNow()}</>
                    ) : (
                      '-'
                    )}
                  </dd>
                  <dt className='sr-only sm:hidden'>Created</dt>
                  <dd className='mt-1 text-gray-700 sm:hidden'>
                    {user?.created ? (
                      <>created {dayjs(user.created).fromNow()}</>
                    ) : (
                      '-'
                    )}
                  </dd>
                </dl>
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:max-w-[10rem] lg:table-cell'>
                {user.providerNames?.map((provider, index) => (
                  <span key={provider}>
                    {provider}
                    {index !== user.providerNames.length - 1 && <>, </>}
                  </span>
                ))}
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:table-cell sm:max-w-[10rem]'>
                <div className='flex items-center py-2'>
                  {user?.lastSeenAt ? (
                    <>{dayjs(user.lastSeenAt).fromNow()}</>
                  ) : (
                    '-'
                  )}
                </div>
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:table-cell sm:max-w-[10rem]'>
                <div className='flex items-center py-2'>
                  {user?.created ? <>{dayjs(user.created).fromNow()}</> : '-'}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function Users() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 13
  const { data: { items, totalPages, totalCount } = {}, error } = useSWR(
    `/api/users?page=${page}&limit=${limit}`
  )
  const { admin, loading: adminLoading } = useAdmin()
  const users = items || []

  const loading = adminLoading || (!users && !error)

  return (
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      <Head>
        <title>Users - Infra</title>
      </Head>
      <div className='py-6'>
        <PageHeader buttonHref={admin && '/users/add'} buttonLabel='User' />
      </div>
      <div className='px-4 sm:px-6 md:px-0'>
        {!loading && (
          <div className='flex flex-1 flex-col space-y-4'>
            {error?.status ? (
              <div className='my-20 text-center text-sm font-light text-gray-300'>
                {error?.info?.message}
              </div>
            ) : (
              <div className='flex min-h-0 flex-1 flex-col px-0 md:px-6 xl:px-0'>
                <UserTable users={users} />
                {users?.length === 0 && page === 1 && (
                  <EmptyTable
                    title='There are no users'
                    subtitle='Invite users to Infra and manage their access.'
                    iconPath='/users.svg'
                    icon={<UsersIcon />}
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
        )}
      </div>
    </div>
  )
}

Users.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
