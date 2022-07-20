import { useRouter } from 'next/router'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/solid'

function getPageNums(selected, count, totalPages) {
  let pageNums = []
  let beginOffset = Math.max(3, 6 + selected - totalPages)
  for (
    let i = Math.max(1, selected - beginOffset);
    pageNums.length <= count && i <= totalPages;
    i++
  ) {
    pageNums.push(i)
  }

  return pageNums
}

function Pages({ path, selected, count, totalPages }) {
  return getPageNums(selected, count, totalPages).map(page => (
    <a
      href={path + '?p=' + page}
      className={`inline-flex items-center py-2 px-4 text-sm font-medium text-gray-500 hover:text-violet-300 ${
        selected === page ? 'rounded-md bg-gray-700' : ''
      }`}
      key={page}
    >
      {page}
    </a>
  ))
}

function LeftArrow({ path, page }) {
  return (
    <a
      href={path + '?p=' + (page > 1 ? page - 1 : 1)}
      className=' inline-flex items-center pr-1 text-sm font-medium text-gray-500 hover:text-violet-300 '
    >
      <ChevronLeftIcon
        className=' h-5 w-5 text-gray-400 hover:text-violet-300'
        aria-hidden='true'
      />
    </a>
  )
}

function RightArrow({ path, page, maxPage }) {
  return (
    <a
      href={path + '?p=' + (page < maxPage ? page + 1 : Math.max(1, maxPage))}
      className='inline-flex items-center pl-1 text-sm font-medium text-gray-500 hover:text-violet-300 '
    >
      <ChevronRightIcon
        className='ml-3 h-5 w-5 text-gray-400 hover:text-violet-300'
        aria-hidden='true'
      />
    </a>
  )
}

export default function Pagination({ curr, totalPages, totalCount }) {
  const router = useRouter()
  const path = router.pathname
  const limit = 13 // TODO: dynamic limit

  curr = curr === undefined ? 1 : parseInt(curr)
  totalPages = totalPages === undefined ? 1 : parseInt(totalPages)
  totalCount = totalCount === undefined ? 0 : parseInt(totalCount)

  const lowerItem = totalCount === 0 ? 0 : 1 + (curr - 1) * limit
  const upperItem = Math.min(1 + (curr - 1) * limit + limit, totalCount)

  return (
    totalPages !== undefined &&
    totalPages > 1 && (
      <div>
        <nav key='paginator' className='flex justify-end px-4 pb-2'>
          <LeftArrow path={path} page={curr}></LeftArrow>
          <Pages
            path={path}
            count={7}
            totalPages={totalPages}
            selected={curr}
          ></Pages>
          <RightArrow path={path} page={curr} maxPage={totalPages}></RightArrow>
        </nav>
        <h3
          key='results'
          className='flex justify-end px-4 pb-4 text-3xs text-gray-400'
        >
          Displaying {lowerItem}–{upperItem} out of {totalCount}
        </h3>
      </div>
    )
  )
}
