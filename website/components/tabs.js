import { createContext, useEffect, useRef } from 'react'
import { useRouter } from 'next/router'

export const TabContext = createContext()

export function useTabGroups() {
  const router = useRouter()

  const groups = {}

  for (const [k, v] of Object.entries(router.query)) {
    if (k.startsWith('group-')) {
      groups[k.replace('group-', '')] = v
    }
  }

  function setGroups(groups = {}) {
    const query = {}
    for (const [k, v] of Object.entries(groups)) {
      query[`group-${k}`] = v
    }

    router.push(
      {
        pathname: router.pathname,
        query: { ...router.query, ...query },
      },
      undefined,
      { shallow: true }
    )
  }

  return [groups, setGroups]
}

export default function Tabs({ labels, children, group = 'default' }) {
  const [groups, setGroups] = useTabGroups()
  const currentTab = groups?.[group] || labels[0]
  const tabsRef = useRef({})
  const activeRef = useRef(null)

  useEffect(() => {
    function resize() {
      const tabElement = tabsRef.current?.[currentTab]
      if (tabElement && activeRef.current) {
        activeRef.current.style.transform = `translateX(${tabElement.offsetLeft}px) translateY(${tabElement.offsetTop}px)`
        activeRef.current.style.width = `${tabElement.offsetWidth}px`
        activeRef.current.style.height = `${tabElement.offsetHeight}px`
      }
    }

    resize()

    window.addEventListener('resize', resize)
    return () => {
      window.removeEventListener('resize', resize)
    }
  }, [currentTab, labels])

  return (
    <TabContext.Provider value={currentTab}>
      <div className='relative flex flex-wrap'>
        <div
          ref={activeRef}
          className={`absolute h-4 w-4 rounded-lg bg-blue-600/5 ${
            activeRef.current && groups?.[group] ? 'transition-all' : ''
          }`}
        ></div>
        {labels.map(label => (
          <button
            key={label}
            ref={e => (tabsRef.current[label] = e)}
            className={`${
              label === currentTab
                ? ' text-blue-700 hover:text-blue-700'
                : 'text-gray-500 hover:text-gray-800'
            } rounded-lg px-7 py-1.5 text-sm font-medium leading-loose transition-colors`}
            onClick={() => {
              setGroups({ ...groups, [group || 'default']: label })
            }}
          >
            {label}
          </button>
        ))}
      </div>
      {children}
    </TabContext.Provider>
  )
}
