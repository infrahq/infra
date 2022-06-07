import Link from 'next/link'
import useSWR from 'swr'

export default function () {
  const { data: auth } = useSWR('/api/users/self')

  return (
    <div className='sm:w-80 lg:w-[500px]'>
      <div className='text-2xs leading-none uppercase text-gray-400 border-b border-gray-800 pb-6'>Account</div>
      <div className='pt-3 flex flex-col space-y-2'>
        <div className='flex group'>
          <div className='flex flex-1 items-center'>
            <div className='text-gray-400 text-2xs w-[26%]'>Email</div>
            <div className='text-2xs'>{auth?.name}</div>
          </div>
        </div>
        <div className='flex group'>
          <div className='flex flex-1 items-center'>
            <div className='text-gray-400 text-2xs w-[30%]'>Password</div>
            <div className='text-2xs'>*****</div>
          </div>
          <div className='flex justify-end'>
            <Link href='/settings/password-reset'>
              <a className='flex-none p-2 -mr-2 cursor-pointer uppercase text-2xs text-gray-500 hover:text-violet-100'>Change</a>
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
