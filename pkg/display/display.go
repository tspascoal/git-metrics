package display

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"git-metrics/pkg/git"
	"git-metrics/pkg/models"
	"git-metrics/pkg/utils"
)

// PrintGrowthTableHeader prints the header for the growth table
func PrintGrowthTableHeader() {
	fmt.Println()
	fmt.Println("HISTORIC & ESTIMATED GROWTH ####################################################################")
	fmt.Println()
	fmt.Println("Year        Commits                  Trees                  Blobs           On-disk size")
	fmt.Println("------------------------------------------------------------------------------------------------")
}

// PrintGrowthTableRow prints a row of the growth table
func PrintGrowthTableRow(statistics, previous models.GrowthStatistics, information models.RepositoryInformation, isEstimate bool, currentYear int) {
	commitsDifference := float64(statistics.Commits-previous.Commits) / float64(information.TotalCommits) * 100
	treesDifference := float64(statistics.Trees-previous.Trees) / float64(information.TotalTrees) * 100
	blobsDifference := float64(statistics.Blobs-previous.Blobs) / float64(information.TotalBlobs) * 100
	compressedDifference := float64(statistics.Compressed-previous.Compressed) / float64(information.CompressedSize) * 100

	yearDisplay := strconv.Itoa(statistics.Year)
	if isEstimate {
		yearDisplay += "*"
	} else if statistics.Year == currentYear {
		fmt.Println("------------------------------------------------------------------------------------------------")
		yearDisplay += "^"
	}

	fmt.Printf("%-5s %13s %+5.0f %%  %13s %+5.0f %%  %13s %+5.0f %%  %13s %+5.0f %%\n",
		yearDisplay,
		utils.FormatNumber(statistics.Commits), commitsDifference,
		utils.FormatNumber(statistics.Trees), treesDifference,
		utils.FormatNumber(statistics.Blobs), blobsDifference,
		utils.FormatSize(statistics.Compressed), compressedDifference)
}

// PrintLargestFiles prints information about the largest files
func PrintLargestFiles(files []models.FileInformation, totalFilesSize int64, totalBlobs int, totalFiles int) {
	fmt.Println("\nLARGEST FILES ##################################################################################")
	fmt.Println()
	fmt.Println("File path                              Last commit          Blobs           On-disk size")
	fmt.Println("------------------------------------------------------------------------------------------------")

	// Track totals for the selected files
	var totalSelectedBlobs int
	var totalSelectedSize int64

	// Calculate total size of all files in repository
	for _, file := range files {
		// Get the last change date for the file
		lastChangeCommand := exec.Command("git", "log", "-1", "--format=%cD", "--", file.Path)
		lastChangeOutput, err := lastChangeCommand.Output()
		if err == nil {
			lastChange, _ := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", strings.TrimSpace(string(lastChangeOutput)))
			file.LastChange = lastChange
		}

		percentageSize := float64(file.CompressedSize) / float64(totalFilesSize) * 100
		percentageBlobs := float64(file.Blobs) / float64(totalBlobs) * 100
		fmt.Printf("%-44s  %s  %13s %5.1f %%  %13s %5.1f %%\n",
			utils.TruncatePath(file.Path, 44),
			file.LastChange.Format("2006"),
			utils.FormatNumber(file.Blobs),
			percentageBlobs,
			utils.FormatSize(file.CompressedSize),
			percentageSize)

		totalSelectedBlobs += file.Blobs
		totalSelectedSize += file.CompressedSize
	}

	// Print separator and selected files totals row
	fmt.Println("------------------------------------------------------------------------------------------------")
	fmt.Printf("%-44s  %s  %13s %5.1f %%  %13s %5.1f %%\n",
		fmt.Sprintf("├─ Top %s", utils.FormatNumber(len(files))),
		"    ",
		utils.FormatNumber(totalSelectedBlobs),
		float64(totalSelectedBlobs)/float64(totalBlobs)*100,
		utils.FormatSize(totalSelectedSize),
		float64(totalSelectedSize)/float64(totalFilesSize)*100)

	// Print grand totals row
	fmt.Printf("%-44s  %s  %13s %5.1f %%  %13s %5.1f %%\n",
		fmt.Sprintf("└─ Out of %s", utils.FormatNumber(totalFiles)),
		"    ",
		utils.FormatNumber(totalBlobs),
		100.0,
		utils.FormatSize(totalFilesSize),
		100.0)
}

// PrintTopFileExtensions prints the top file extensions by size
func PrintTopFileExtensions(blobs []models.FileInformation, totalBlobs int, totalSize int64) {
	extensionStatistics := make(map[string]struct {
		size       int64
		filesCount int
		blobsCount int
	})
	for _, blob := range blobs {
		extension := filepath.Ext(blob.Path)
		if extension == "" {
			extension = "No Extension"
		}
		statistics := extensionStatistics[extension]
		statistics.size += blob.CompressedSize
		statistics.filesCount++
		statistics.blobsCount += blob.Blobs
		extensionStatistics[extension] = statistics
	}

	// Create a slice for sorting.
	type extensionStatistic struct {
		extension  string
		size       int64
		filesCount int
		blobsCount int
	}
	var statistics []extensionStatistic
	for extension, statistic := range extensionStatistics {
		statistics = append(statistics, extensionStatistic{
			extension:  extension,
			size:       statistic.size,
			filesCount: statistic.filesCount,
			blobsCount: statistic.blobsCount,
		})
	}
	sort.Slice(statistics, func(i, j int) bool {
		return statistics[i].size > statistics[j].size
	})

	// Calculate totals from all extensions first
	var totalExtFilesCount, totalExtBlobsCount int
	var totalExtSize int64
	for _, statistic := range extensionStatistics {
		totalExtFilesCount += statistic.filesCount
		totalExtBlobsCount += statistic.blobsCount
		totalExtSize += statistic.size
	}

	// Limit to top 10
	if len(statistics) > 10 {
		statistics = statistics[:10]
	}

	// Track totals for displayed extensions (top 10)
	var selectedFilesCount int
	var selectedBlobsCount int
	var selectedSize int64

	// Display results.
	fmt.Println("\nLARGEST FILE EXTENSIONS ########################################################################")
	fmt.Println()
	fmt.Println("Extension                            Files                  Blobs           On-disk size")
	fmt.Println("------------------------------------------------------------------------------------------------")
	for _, statistic := range statistics {
		percentageFiles := float64(statistic.filesCount) / float64(totalExtFilesCount) * 100
		percentageBlobs := float64(statistic.blobsCount) / float64(totalBlobs) * 100
		percentageSize := float64(statistic.size) / float64(totalSize) * 100
		fmt.Printf("%-28s %13s %5.1f %%  %13s %5.1f %%  %13s %5.1f %%\n",
			statistic.extension, utils.FormatNumber(statistic.filesCount), percentageFiles, utils.FormatNumber(statistic.blobsCount), percentageBlobs, utils.FormatSize(statistic.size), percentageSize)

		selectedFilesCount += statistic.filesCount
		selectedBlobsCount += statistic.blobsCount
		selectedSize += statistic.size
	}

	// Print separator and top 10 totals row
	fmt.Println("------------------------------------------------------------------------------------------------")
	fmt.Printf("%-28s %13s %5.1f %%  %13s %5.1f %%  %13s %5.1f %%\n",
		fmt.Sprintf("├─ Top %s", utils.FormatNumber(len(statistics))),
		utils.FormatNumber(selectedFilesCount),
		float64(selectedFilesCount)/float64(totalExtFilesCount)*100,
		utils.FormatNumber(selectedBlobsCount),
		float64(selectedBlobsCount)/float64(totalExtBlobsCount)*100,
		utils.FormatSize(selectedSize),
		float64(selectedSize)/float64(totalExtSize)*100)

	// Print grand totals row using full totals
	fmt.Printf("%-28s %13s %5.1f %%  %13s %5.1f %%  %13s %5.1f %%\n",
		fmt.Sprintf("└─ Out of %s", utils.FormatNumber(len(extensionStatistics))),
		utils.FormatNumber(totalExtFilesCount),
		100.0, // Always 100% for totals
		utils.FormatNumber(totalExtBlobsCount),
		100.0,
		utils.FormatSize(totalExtSize),
		100.0)
}

// PrintMachineInfo prints information about the system
func PrintMachineInformation() {
	fmt.Println()
	fmt.Println("RUN ############################################################################################")
	fmt.Println()
	fmt.Printf("Start time                 %s\n", time.Now().Format("Mon, 02 Jan 2006 15:04 MST"))
	fmt.Printf("Machine                    %d CPU cores with %d GB memory (%s on %s)\n",
		runtime.NumCPU(),
		utils.GetMemoryInGigabytes(),
		utils.GetOperatingSystemInformation(),
		utils.GetChipInformation())
	fmt.Printf("Git version                %s\n", git.GetGitVersion())
}
