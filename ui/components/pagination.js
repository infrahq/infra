import { useRouter } from 'next/router'
import Link from 'next/link'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/solid'

export function Pages({ path, selected, count, totalPages }) {
  const pages = []
  const beginOffset = Math.max(
    Math.floor(count / 2),
    count - 1 + selected - totalPages
  )
  for (
    let page = Math.max(1, selected - beginOffset);
    pages.length < count && page <= totalPages;
    page++
  ) {
    pages.push(
      <Link
        key={page}
        data-testid='pages-button-link'
        href={path + '?p=' + page}
      >
        <a className='relative hidden items-center border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-500 hover:bg-gray-100 focus:z-20 md:inline-flex'>
          {page}
        </a>
      </Link>
    )
  }

  return pages.map(page => page)
}

export function Arrow({ path, direction }) {
  return (
    <>
      {direction === 'LEFT' && (
        <Link href={path} data-testid={`${direction}-arrow-button-link`}>
          <a className='relative inline-flex items-center rounded-l-md border border-gray-300 bg-white px-2 py-2 text-sm font-medium text-gray-500 hover:bg-gray-100 focus:z-20'>
            <span className='sr-only'>Previous</span>
            <ChevronLeftIcon className='h-5 w-5' aria-hidden='true' />
          </a>
        </Link>
      )}
      {direction === 'RIGHT' && (
        <Link href={path} data-testid={`${direction}-arrow-button-link`}>
          <a className='relative inline-flex items-center rounded-r-md border border-gray-300 bg-white px-2 py-2 text-sm font-medium text-gray-500 hover:bg-gray-100 focus:z-20'>
            <span className='sr-only'>Next</span>
            <ChevronRightIcon className='h-5 w-5' aria-hidden='true' />
          </a>
        </Link>
      )}
    </>
  )
}

export default function Pagination({
  curr = 1,
  totalPages = 0,
  totalCount = 0,
}) {
  const router = useRouter()
  const path = router.pathname
  const limit = 13 // TODO: dynamic limit

  curr = parseInt(curr)
  totalPages = parseInt(totalPages)
  totalCount = parseInt(totalCount)

  const lowerItem = totalCount === 0 ? 0 : 1 + (curr - 1) * limit
  const upperItem = Math.min(curr * limit, totalCount)

  return (
    <div className='flex items-center justify-between bg-white px-4 py-3 sm:px-6'>
      <div className='flex flex-1 justify-between sm:hidden'>
        <Link href={path + '?p=' + (curr > 1 ? curr - 1 : 1)}>
          <a className='relative inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100'>
            Previous
          </a>
        </Link>
        <Link
          href={
            path +
            '?p=' +
            (curr < totalPages ? curr + 1 : Math.max(1, totalPages))
          }
        >
          <a className='relative ml-3 inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100'>
            Next
          </a>
        </Link>
      </div>
      <div className='hidden sm:flex sm:flex-1 sm:items-center sm:justify-between'>
        <div>
          <p className='text-sm text-gray-700'>
            Showing <span className='font-medium'>{lowerItem}</span> to{' '}
            <span className='font-medium'>{upperItem}</span> of{' '}
            <span className='font-medium'>{totalCount}</span> results
          </p>
        </div>
        <div>
          <nav
            className='isolate inline-flex -space-x-px rounded-md shadow-sm'
            aria-label='Pagination'
          >
            <Arrow
              direction='LEFT'
              path={path + '?p=' + (curr > 1 ? curr - 1 : 1)}
            ></Arrow>
            <Pages
              path={path}
              count={7}
              totalPages={totalPages}
              selected={curr}
            ></Pages>
            <Arrow
              direction='RIGHT'
              path={
                path +
                '?p=' +
                (curr < totalPages ? curr + 1 : Math.max(1, totalPages))
              }
            ></Arrow>
          </nav>
        </div>
      </div>
    </div>
  )
}
