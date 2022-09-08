import Head from 'next/head'
import useSWR from 'swr'
import { useRouter } from 'next/router'
import { UserGroupIcon } from '@heroicons/react/outline'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import Breadcrumbs from '../../components/breadcrumbs'
import Dashboard from '../../components/layouts/dashboard'
import EmptyTable from '../../components/empty-table'
import Pagination from '../../components/pagination'
import PageHeader from '../../components/page-header'

function GroupTable({ groups }) {
  const router = useRouter()

  return (
    <div className='overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg'>
      <Breadcrumbs>Groups</Breadcrumbs>
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
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 sm:table-cell'
            >
              Users
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
          {groups?.map(group => (
            <tr
              key={group.id}
              onClick={() => router.replace(`/groups/${group.id}`)}
              className='hover:cursor-pointer hover:bg-gray-100'
            >
              <td className='w-full max-w-0 py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:w-auto sm:max-w-none sm:pl-6'>
                <div className='flex items-center py-1.5'>
                  <div className='text-sm sm:max-w-[10rem]'>{group.name}</div>
                </div>
                <dl className='font-normal sm:hidden'>
                  <dt className='sr-only sm:hidden'>Users</dt>
                  <dd className='mt-1 text-gray-700 sm:hidden'>
                    <>
                      {group.totalUsers === undefined ? (
                        '-'
                      ) : (
                        <>
                          {group.totalUsers}{' '}
                          {group.totalUsers === 1 ? 'member' : 'members'}
                        </>
                      )}
                    </>
                  </dd>
                  <dt className='sr-only sm:hidden'>Created</dt>
                  <dd className='mt-1 text-gray-700 sm:hidden'>
                    {group?.created ? (
                      <>created {dayjs(group.created).fromNow()}</>
                    ) : (
                      '-'
                    )}
                  </dd>
                </dl>
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:table-cell sm:max-w-[10rem]'>
                <div className='flex items-center py-2'>
                  {group.totalUsers === undefined ? (
                    '-'
                  ) : (
                    <>
                      {group.totalUsers}{' '}
                      {group.totalUsers === 1 ? 'member' : 'members'}
                    </>
                  )}
                </div>
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:table-cell sm:max-w-[10rem]'>
                <div className='flex items-center py-2'>
                  {group?.created ? <>{dayjs(group.created).fromNow()}</> : '-'}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function Groups() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 13
  const { data: { items: groups, totalPages, totalCount } = {}, error } =
    useSWR(`/api/groups?page=${page}&limit=${limit}`)
  const { admin, loading: adminLoading } = useAdmin()

  const loading = adminLoading || (!groups && !error)

  return (
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      <Head>
        <title>Groups - Infra</title>
      </Head>
      <div className='py-6'>
        <PageHeader buttonHref={admin && '/groups/add'} buttonLabel='Group' />
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
                <GroupTable groups={groups} />
                {groups?.length === 0 && (
                  <EmptyTable
                    title='There are no groups'
                    subtitle='Connect, create and manage your groups.'
                    iconPath='/groups.svg'
                    icon={<UserGroupIcon />}
                  />
                )}
                {totalPages > 1 && (
                  <Pagination
                    curr={page}
                    totalPages={totalPages}
                    totalCount={totalCount}
                  ></Pagination>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

Groups.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
