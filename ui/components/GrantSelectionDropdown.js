export default ({
  optionType,
  options,
  handleChangeSelection,
  selectedValue
}) => {
  return (
    <select
      id={optionType}
      name={optionType}
      className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none sm:text-sm bg-transparent'
      defaultValue={selectedValue}
      onChange={handleChangeSelection}
    >
      {options.map((option) => (
        <option key={option} value={option}>{option}</option>
      ))}
    </select>
  )
}
