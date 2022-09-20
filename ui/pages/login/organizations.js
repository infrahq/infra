import Link from 'next/link'
import Cookies from 'universal-cookie'
import { useRouter } from 'next/router'
import { ChevronRightIcon } from '@heroicons/react/outline'

import LoginLayout from '../../components/layouts/login'

export default function Organizations() {
  const cookies = new Cookies()
  const organizations = cookies.get('orgs')
  const router = useRouter()

  if (!organizations?.length) {
    router.replace('/forgot-domain')
    return null
  }

  return (
    <div className='flex min-h-[280px] w-full flex-col items-center px-10 py-10'>
      <h1 className='font-display text-xl font-semibold leading-snug'>
        Log in to Infra
      </h1>
      <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
        Choose an organization to log in to.
      </h2>
      <div className='my-6 w-full max-w-[240px] flex-1'>
        {organizations?.map(o => (
          <a
            href={`//${o.url}`}
            key={o.url}
            title={`${o.name} — ${o.url}`}
            className='group my-2 flex w-full items-center justify-between
             rounded-md border border-gray-300 bg-white py-2.5 px-4
              hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'
          >
            <div className='truncate text-left'>
              <div className='truncate text-sm leading-snug'>{o.name}</div>
              <div className='truncate text-xs text-gray-500'>{o.url}</div>
            </div>
            <div>
              <ChevronRightIcon className='ml-2 mt-0.5 h-3 w-3 flex-none stroke-2 group-hover:text-gray-400' />
            </div>
          </a>
        ))}
      </div>
      <p className='text-center text-xs text-gray-500'>
        Not seeing your organization? <br />
        <Link href='/forgot-domain'>
          <a className='my-1 inline-flex items-center font-semibold text-blue-500'>
            Find my organization{' '}
            <ChevronRightIcon className='mt-0.5 h-3 w-3 stroke-2' />
          </a>
        </Link>
      </p>
    </div>
  )
}

Organizations.layout = page => <LoginLayout>{page}</LoginLayout>
